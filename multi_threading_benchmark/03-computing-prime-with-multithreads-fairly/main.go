/*
	In this program, we try to keep all threads busy for an equal amount of time by including a variable that holds the current value
	for which prime is to be found for any thread that picks it. This way each thread is constantly working on something and is not
	free comparing to us batching the threads in the unfair approach

	Why are we doing this ?
	Cause with respect to finding primes, there are lesser primes as the range of values increase. You find more  prime numbers near the
	lower end region compared to the higher end region and for the reason that as we calculate till square root of a number, square roots for
	larger numbers as we go high are bigger than smaller numbers at the lower end region.
	Hence the threads working on the lower end region during batching seem to finish
	faster than the ones on the higher ends as they find more prime numbers (need not traverse till sqrt of n for most numbers)



	Result of example run on my Macbook M1: (The numbers you get on your OS could vary based on different factors like CPU and Processor configuration, currently available RAM etc)

	Time taken by thread : 0 is 7.6395865s
	Time taken by thread : 8 is 7.607885834s
	Time taken by thread : 3 is 7.6396615s
	Time taken by thread : 2 is 7.638764375s
	Time taken by thread : 9 is 7.639679167s
	Time taken by thread : 7 is 7.628969417s
	Time taken by thread : 1 is 7.639678292s
	Time taken by thread : 4 is 7.639448167s
	Time taken by thread : 6 is 7.639510833s
	Time taken by thread : 5 is 7.639443792s

	Checking till 100000000 found 5761455  prime numbers. Time taken :  7.639898042s
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
var currentNumber int32 = 2

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

func doWork(threadName string, wg *sync.WaitGroup) {
	startTimeForThread := time.Now()

	defer wg.Done()

	for {
		numberToBeChecked := atomic.AddInt32(&currentNumber, 1)
		if numberToBeChecked > int32(MAX_INT) {
			break
		}

		checkPrime(int(numberToBeChecked))
	}

	fmt.Printf("Time taken by thread : %s is %s\n", threadName, time.Since(startTimeForThread))
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

	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go doWork(strconv.Itoa(i), &wg)
	}

	wg.Wait()

	// Doing [totalPrimeNumbers+1] to include 2(the only even prime) which we did not account for
	fmt.Println("Checking till", MAX_INT, "found", totalPrimeNumbers+1, " prime numbers. Time taken : ", time.Since(startTime))
}
