package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	wg := &sync.WaitGroup{}

	for _, j := range jobs {
		out = make(chan interface{})
		wg.Add(1)

		go func(jobFunc job, in, out chan interface{}, wg *sync.WaitGroup) {
			defer close(out)
			defer wg.Done()
			jobFunc(in, out)
		}(j, in, out, wg)

		in = out
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	mu := &sync.Mutex{}

	for data := range in {
		wg.Add(1)
		go func(data int) {
			defer wg.Done()

			// DataSignerCrc32(data)
			crc32ResultCh := make(chan string)
			go func() {
				defer close(crc32ResultCh)
				crc32ResultCh <- DataSignerCrc32(strconv.Itoa(data))
			}()

			// DataSignerMd5(data)
			mu.Lock()
			md5Result := DataSignerMd5(strconv.Itoa(data))
			mu.Unlock()

			// DataSignerCrc32(md5(data))
			crc32Md5ResultCh := make(chan string)
			go func() {
				defer close(crc32Md5ResultCh)
				crc32Md5ResultCh <- DataSignerCrc32(md5Result)
			}()

			// Combine results
			crc32Result := <-crc32ResultCh
			crc32Md5Result := <-crc32Md5ResultCh
			result := crc32Result + "~" + crc32Md5Result
			out <- result
		}(data.(int))
	}

	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup

	for data := range in {
		wg.Add(1)
		go func(data string) {
			defer wg.Done()

			hashResults := make([]string, 6)
			hashWg := &sync.WaitGroup{}

			for th := 0; th < 6; th++ {
				hashWg.Add(1)
				go func(th int, data string) {
					defer hashWg.Done()
					hash := DataSignerCrc32(strconv.Itoa(th) + data)
					hashResults[th] = hash
				}(th, data)
			}

			hashWg.Wait()
			result := strings.Join(hashResults, "")
			out <- result
		}(data.(string))
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for result := range in {
		results = append(results, result.(string))
	}

	sort.Strings(results)
	out <- strings.Join(results, "_")
}
