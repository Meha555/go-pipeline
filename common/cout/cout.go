package cout

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ElapsedSecondsDummyJob 生成一个临时 goroutine 来打印进度条
func ElapsedSecondsDummyJob(max, min uint8) {
	fmt.Printf("可能的耗时时间在[%d, %d], 单位: 秒\n", max, min)
	elapsed := rand.Intn(int(max-min+1)) + int(min)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(elapsed)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Print(".")
			case <-ctx.Done():
				fmt.Println()
				return
			}
		}
	}()
	// FIXME 这里实现的有问题
	time.Sleep(time.Duration(elapsed) * time.Second)
	wg.Wait()
}
