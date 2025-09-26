package notify

type Notifier interface {
	Notify() error
}