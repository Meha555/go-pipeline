package common

type OptString string

func (o *OptString) ValueOr(str string) string {
	if *o == "" {
		*o = OptString(str)
	}
	return string(*o)
}

// InvokeForEach 对输入范围中的每个元素调用指定函数
func InvokeForEach[Rng ~[]T, T any, Func func(...any), Args any](rangeData Rng, fn Func, args ...Args) {
	for _, item := range rangeData {
		params := make([]any, len(args)+1)
		for i, arg := range args {
			params[i] = arg
		}
		params[len(args)] = item
		fn(params...)
	}
}