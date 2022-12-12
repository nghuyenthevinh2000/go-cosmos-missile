package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
)

type Target struct {
	url    string
	action string
}

func attack(attackers int, total_launch uint32) {
	var wg sync.WaitGroup
	finish_batch := int32(0)
	registered_new_height := make([]int64, attackers)
	attackers_continue := make([]bool, attackers)
	os.Mkdir("attackers", os.ModePerm)

	// Query attackers
	target := Target{
		url:    "http://localhost:26657/",
		action: "",
	}

	for i := 0; i < attackers; i++ {
		wg.Add(1)

		// https://stackoverflow.com/questions/40326723/go-vet-range-variable-captured-by-func-literal-when-using-go-routine-inside-of-f
		go func(attacker int) {
			f, err := os.Create(fmt.Sprintf("attackers/attacker_%d.txt", attacker))
			if err != nil {
				panic(err.Error())
			}

			defer func() {
				wg.Done()
				f.Close()
			}()

			for launch_time := uint32(0); launch_time < total_launch; launch_time++ {
				if finish_batch == int32(attackers) {
					// reset counter
					atomic.StoreInt32(&finish_batch, 0)
					attackers_continue = make([]bool, attackers)

					// find most occurences
					occurence := make(map[int64]int)
					most_occurence := registered_new_height[0]
					for _, height := range registered_new_height {
						if _, ok := occurence[height]; !ok {
							occurence[height] = 0
						}

						occurence[height] += 1
						if occurence[height] > occurence[most_occurence] {
							most_occurence = height
						}
					}

					// statistic
					fmt.Printf("most occurence is = %d with frequency = %d \n", most_occurence, occurence[most_occurence])
					fmt.Printf("missed one = %d \n", attackers-occurence[most_occurence])
				}

				if attackers_continue[attacker] {
					time.Sleep(time.Second)
					continue
				}

				new := fire(f, target)
				// mark as completed
				atomic.AddInt32(&finish_batch, 1)
				registered_new_height[attacker] = new
				attackers_continue[attacker] = true
			}
		}(i)
	}

	wg.Wait()
}

func fire(f *os.File, target Target) int64 {
	tmClient, err := client.NewClientFromNode(target.url)
	if err != nil {
		panic(err.Error())
	}
	res, err := tmClient.Status(context.Background())
	if err != nil {
		panic(err.Error())
	}

	_, err = f.WriteString(fmt.Sprintf("res = %d \n", res.SyncInfo.LatestBlockHeight))
	if err != nil {
		panic(err.Error())
	}

	f.Sync()

	return res.SyncInfo.LatestBlockHeight
}
