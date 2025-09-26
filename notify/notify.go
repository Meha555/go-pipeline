package notify

// type Message struct {
// 	Title string
// 	Content string
// }

// type Notifier interface {
// 	Notify(msg Message) error
// }

// NOTE 由于email\bot\sms协议不一致，所以不可能做到一个统一的Message结构体作为输入，然后不同的notifier实现完成不同的发送逻辑
// 而如果Message也是一个接口，那么就表示notifier内部只能调用Message的接口方法，显然做不到——因为具体的notifier要求按照协议的具体message，那么就相当于需要传入具体的message，不存在可以抽象的空间
// 所以这里做不到一个统一的notify接口
