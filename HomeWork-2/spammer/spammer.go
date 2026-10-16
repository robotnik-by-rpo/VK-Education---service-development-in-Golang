package main

import (
	"fmt"
	"sort"
	"sync"
)

func RunPipeline(cmds ...cmd) {

	in := make(chan interface{})
	close(in)

	wg := sync.WaitGroup{}
	for _, c := range cmds{
		out := make(chan interface{})
		wg.Add(1)
		go func(c cmd, in, out chan interface{}){
			defer wg.Done()
			defer close(out)
			c(in , out)
		}(c, in, out)

		in = out
	}
	wg.Wait()
}

func SelectUsers(in, out chan interface{}) {
	// 	in - string
	// 	out - User
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	seen := make(map[uint64]struct{})
	for emailRaw := range in {
		email, ok := emailRaw.(string) 
		if !ok{
			continue
		}

		wg.Add(1)
		go func(emaiIn string) {
			defer wg.Done()
			user := GetUser(emaiIn)
			
			mu.Lock()
			_, exist := seen[user.ID]
			if !exist{
				seen[user.ID] = struct{}{}
			}
			mu.Unlock()

			if !exist{
				out <- user
			}
			
		}(email)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	// 	in - User
	// 	out - MsgID
	batch := make([]User, 0, GetMessagesMaxUsersBatch)
	wg := sync.WaitGroup{}

	flush := func() {
		if len(batch) == 0{
			return
		}

		users := make([]User, len(batch))
		copy(users, batch)
		batch = batch[:0]

		wg.Add(1)

		go func(users []User){
			defer wg.Done()
			msgs, err := GetMessages(users...)
			if err != nil{
				return
			}

			for _, m := range msgs{
				out <- m
			}
		}(users)
	}

	for userRaw := range in{
		user, ok := userRaw.(User)
		if !ok {
			continue
		}

		batch = append(batch, user)
		if len(batch) == GetMessagesMaxUsersBatch{
			flush()
		}
	}	
	flush()

	wg.Wait()
}

func CheckSpam(in, out chan interface{}) {
	// in - MsgID
	// out - MsgData

	wg := sync.WaitGroup{}
    for i := 0; i < HasSpamMaxAsyncRequests; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for msgIn := range in {
                msgID, ok := msgIn.(MsgID)
                if !ok {
                    continue
                }
                res, err := HasSpam(msgID)
                if err != nil {
                    continue
                }
                out <- MsgData{ID: msgID, HasSpam: res}
            }
        }()
    }

    wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	// in - MsgData
	// out - string

	var results []MsgData
	for d := range in{
		data, ok := d.(MsgData)
		if !ok {
			continue
		}

		results = append(results, data)
	}


	sort.Slice(results, func(i, j int) bool {
		if results[i].HasSpam != results[j].HasSpam{
			return results[i].HasSpam
		}
		return results[i].ID < results[j].ID
	})

	for _, r := range results{
		out <- fmt.Sprintf("%v %v", r.HasSpam, r.ID)
	}
}