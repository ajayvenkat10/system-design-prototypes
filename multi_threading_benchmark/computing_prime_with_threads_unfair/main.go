/*
	In this program we try to minimize the time taken to find prime numbers between 3 and INT_MAX by introducing threads.

	We take 10 threads and divide the range by the number of threads to get the batch of numbers that each thread has to process.
	We can see that there is a significant difference in the time taken by this program compared to running on the main thread.



	Result of example run on my Macbook M1: (The numbers you get on your OS could vary based on different factors like CPU and Processor configuration, currently available RAM etc)

	Time taken by thread : 0 for tha batch 3 to 10000003 is 2.870186708s
	Time taken by thread : 1 for tha batch 10000003 to 20000003 is 4.305794625s
	Time taken by thread : 2 for tha batch 20000003 to 30000003 is 5.104312417s
	Time taken by thread : 3 for tha batch 30000003 to 40000003 is 5.749074667s
	Time taken by thread : 4 for tha batch 40000003 to 50000003 is 6.159044958s
	Time taken by thread : 5 for tha batch 50000003 to 60000003 is 6.680493084s
	Time taken by thread : 6 for tha batch 60000003 to 70000003 is 6.833853375s
	Time taken by thread : 7 for tha batch 70000003 to 80000003 is 7.197813208s
	Time taken by thread : 8 for tha batch 80000003 to 90000003 is 7.375215125s
	Time taken by thread : 9 for tha batch 90000003 to 100000000 is 7.482081334s

	Checking till 100000000 found 5761455  prime numbers. Time taken : 7.482181208s

*/

package main

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var MAX_INT int = 100000000
var CONCURRENCY int = 10
var totalPrimeNumbers int32 = 0

func checkPrime(num int) {
	if num&1 == 0 {
		return
	}

	for i := 3; i <= int(math.Sqrt(float64(num))); i++ {
		if num%i == 0 {
			return
		}
	}

	// Using atomic add as count++ is not thread safe. There are chances of race conditions occurring where there can be inconsistent values after ++ by many threads on a value.
	atomic.AddInt32(&totalPrimeNumbers, 1)
}

func doBatch(threadName string, wg *sync.WaitGroup, nstart, nend int) {
	defer wg.Done()

	startTimeForThread := time.Now()

	for i := nstart; i < nend; i++ {
		checkPrime(i)
	}

	fmt.Printf("Time taken by thread : %s for the batch %d to %d is %s\n", threadName, nstart, nend, time.Since(startTimeForThread))
}

func main() {
	startTime := time.Now()

	/*
		Using WaitGroups for synchronization of threads.
		1. Before starting a goroutine, you increment the WaitGroup counter using the Add method. This informs the WaitGroup that a goroutine is about to start.
		3. Once the goroutine is completed, you need to call the Done() to signal the completion. This will decrement the counter behind the scenes
		4. Outside of this you call the Wait() which is similar to the await. Your program waits for the goroutines in the waitgroup to complete execution.
		5. To understand better, wait is called on the waitgroup and you program waits there until all go routines in the waitgroup have called Done() and the counter is 0.
	*/
	var wg sync.WaitGroup

	nStart := 3
	batchSize := int(float64(MAX_INT) / float64(CONCURRENCY))

	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)

		go doBatch(strconv.Itoa(i), &wg, nStart, func() int {
			if i == CONCURRENCY-1 {
				return MAX_INT
			}
			return nStart + batchSize
		}())
		nStart += batchSize
	}

	wg.Wait()

	// Doing [totalPrimeNumbers+1] to include 2(the only even prime) which we did not account for
	fmt.Println("Checking till", MAX_INT, "found", totalPrimeNumbers+1, " prime numbers. Time taken : ", time.Since(startTime))
}
