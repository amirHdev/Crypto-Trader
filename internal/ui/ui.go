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
	// Settings
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
		// Note: Save to .env? For simplicity, update in memory, persist if needed.
		dialog.ShowInformation("Saved", "Settings updated (in memory)", w)
		tg = notifier.New(cfg) // update notifier
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
	// Create Order
	coinName := widget.NewEntry()
	walletPerc := widget.NewEntry()
	sellPerc := widget.NewEntry()
	date := widget.NewEntry()      // Use fyne-x/widget for date picker if needed, or text YYYY-MM-DD
	timeEntry := widget.NewEntry() // HH:MM:SS
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
	// Recent Orders
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
	refreshRecent()
	recent := widget.NewListWithData(recentList,
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewLabel(""), widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), nil))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			str, _ := i.(binding.String).Get()
			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(str)
			btn := o.(*fyne.Container).Objects[1].(*widget.Button)
			btn.OnTapped = func() {
				// Extract ID from str? Or better, use index
				orders, _ := store.GetAllOrders()
				id := orders[recentList.Index(i)].ID()
				sched.DeleteOrder(id)
				refreshRecent()
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
