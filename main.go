package main

import (
	"component-generator/cmd"
)

func main() {
	// c := make(chan struct{}, 2)
	//
	// wg := &sync.WaitGroup{}
	// wg.Add(1)
	//
	// go func() {
	// 	i := 0
	// 	for {
	// 		i++
	// 		select {
	// 		case msg, ok := <-c:
	// 			fmt.Println("1", msg, ok)
	// 		}
	// 		fmt.Println("reload")
	// 		if i == 10 {
	// 			break
	// 		}
	// 	}
	//
	// 	wg.Done()
	// }()
	//
	// c <- struct{}{}
	// c <- struct{}{}
	// c <- struct{}{}
	//
	// close(c)
	// c <- struct{}{}
	//
	// fmt.Println("done")
	// wg.Wait()
	// return

	cmd.Execute()
}
