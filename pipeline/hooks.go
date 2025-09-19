package pipeline

import "context"

// Hooks 是 Job 的钩子函数，可以指定在某个job之前或之后执行（不论job是否失败）。
// 其本身允许失败，失败时只是终止同级的剩余hooks执行，不会影响所在job的执行。
type Hooks struct {
	Before []*Action
	After  []*Action
}

func doHooks(ctx context.Context, hooks []*Action) (err error) {
	for _, hook := range hooks {
		err = hook.Exec(ctx)
		if err != nil {
			return
		}
	}
	return
}

func (h *Hooks) DoBefore(ctx context.Context) error {
	return doHooks(ctx, h.Before)
}

func (h *Hooks) DoAfter(ctx context.Context) error {
	return doHooks(ctx, h.After)
}
