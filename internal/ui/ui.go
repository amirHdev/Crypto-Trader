package ui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/amirhdev/crypto-trader/internal/config"
	"github.com/amirhdev/crypto-trader/internal/domain"
	"github.com/amirhdev/crypto-trader/internal/notifier"
	"github.com/amirhdev/crypto-trader/internal/scheduler"
	"github.com/amirhdev/crypto-trader/internal/storage"
	"github.com/spf13/viper"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func Run(ctx context.Context, sched *scheduler.Scheduler, store *storage.Storage, cfg *config.Config, tg *notifier.Telegram) {
	a := app.New()
	w := a.NewWindow("Crypto Trader")

	telegramToken := widget.NewEntry()
	telegramToken.SetText(cfg.Telegram.BotToken)
	kucoinKey := widget.NewEntry()
	kucoinKey.SetText(cfg.KuCoin.Key)
	kucoinSecret := widget.NewEntry()
	kucoinSecret.SetText(cfg.KuCoin.Secret)
	kucoinPass := widget.NewEntry()
	kucoinPass.SetText(cfg.KuCoin.Passphrase)
	saveSettings := widget.NewButton("Save Settings", func() {
		cfg.Telegram.BotToken = telegramToken.Text
		cfg.KuCoin.Key = kucoinKey.Text
		cfg.KuCoin.Secret = kucoinSecret.Text
		cfg.KuCoin.Passphrase = kucoinPass.Text

		viper.Set("TELEGRAM_BOT_TOKEN", cfg.Telegram.BotToken)
		viper.Set("KUCOIN_API_KEY", cfg.KuCoin.Key)
		viper.Set("KUCOIN_API_SECRET", cfg.KuCoin.Secret)
		viper.Set("KUCOIN_API_PASSPHRASE", cfg.KuCoin.Passphrase)

		if err := viper.WriteConfig(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save .env: %w", err), w)
			return
		}

		tg = notifier.New(cfg)
		dialog.ShowInformation("Saved", "Settings saved to .env", w)
	})

	settings := container.NewVBox(
		widget.NewLabel("Telegram Token"),
		telegramToken,
		widget.NewLabel("KuCoin API Key"),
		kucoinKey,
		widget.NewLabel("KuCoin Secret"),
		kucoinSecret,
		widget.NewLabel("KuCoin Passphrase"),
		kucoinPass,
		saveSettings,
	)

	recentList := binding.NewStringList()
	refreshRecent := func() {
		orders, err := store.GetAllOrders()
		if err != nil {
			return
		}
		var strs []string
		for _, o := range orders {
			strs = append(strs, fmt.Sprintf("%s | Wallet: %.2f%% | Sell: %.2f%% | At: %s", o.Coin, o.WalletPerc, o.SellPerc, o.ExecuteAt.Format(time.RFC3339)))
		}
		recentList.Set(strs)
	}

	// Create Order
	coinName := widget.NewEntry()
	walletPerc := widget.NewEntry()
	sellPerc := widget.NewEntry()
	date := widget.NewEntry()
	timeEntry := widget.NewEntry()
	addOrder := widget.NewButton("Add Order", func() {
		wp, err1 := strconv.ParseFloat(walletPerc.Text, 64)
		sp, err2 := strconv.ParseFloat(sellPerc.Text, 64)
		if err1 != nil || err2 != nil || wp <= 0 || wp > 100 || sp <= 0 {
			dialog.ShowError(fmt.Errorf("Invalid percentages"), w)
			return
		}
		timestamp, err := time.Parse("2006-01-02 15:04:05", date.Text+" "+timeEntry.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Invalid date/time"), w)
			return
		}
		if timestamp.Before(time.Now()) {
			dialog.ShowError(fmt.Errorf("Time in past"), w)
			return
		}
		ord := domain.Order{
			Coin:       coinName.Text,
			WalletPerc: wp,
			SellPerc:   sp,
			ExecuteAt:  timestamp,
			CreatedAt:  time.Now(),
		}
		if err := sched.AddOrder(ord); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Success", "Order added", w)
		refreshRecent()
	})

	createOrder := container.NewVBox(
		widget.NewLabel("Coin Name (e.g. BTC-USDT)"),
		coinName,
		widget.NewLabel("Wallet % Buy (0-100)"),
		walletPerc,
		widget.NewLabel("Sell % Increase"),
		sellPerc,
		widget.NewLabel("Date (YYYY-MM-DD)"),
		date,
		widget.NewLabel("Time (HH:MM:SS)"),
		timeEntry,
		addOrder,
	)

	refreshRecent()
	recent := widget.NewList(
		func() int {
			ords, _ := store.GetAllOrders()
			return len(ords)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, nil,
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
				widget.NewLabel(""),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			ords, err := store.GetAllOrders()
			if err != nil || i >= len(ords) {
				return
			}
			ord := ords[i]

			cont := o.(*fyne.Container)
			cont.Objects[0].(*widget.Label).SetText(
				fmt.Sprintf("%s | Buy: %.1f%% | Sell+: %.1f%% | %s",
					ord.Coin, ord.WalletPerc, ord.SellPerc,
					ord.ExecuteAt.Format("2006-01-02 15:04"),
				))

			btn := cont.Objects[1].(*widget.Button)
			btn.OnTapped = func() {
				if err := sched.DeleteOrder(ord.ID()); err != nil {
					dialog.ShowError(err, w)
				} else {
					refreshRecent()
				}
			}
		},
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Create Order", createOrder),
		container.NewTabItem("Recent Orders", recent),
		container.NewTabItem("Settings", settings),
	)

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
