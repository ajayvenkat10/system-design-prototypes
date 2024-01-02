/*
	In this program we try to find the number of prime numbers between 3 to INT_MAX (defined in the program).


	Result of example run on my Macbook M1: (The numbers you get on your OS could vary based on different factors like CPU and Processor configuration, currently available RAM etc)

	Checking till 100000000 found 5761455 prime numbers. Time taken : 29.826874709s
*/

package main

import (
	"fmt"
	"math"
	"time"
)

var MAX_INT int = 100000000
var totalPrimeNumbers int = 0

func checkPrime(num int) {
	// There are no even prime numbers apart from 2, hence skipping them
	if num&1 == 0 {
		return
	}

	for i := 3; i <= int(math.Sqrt(float64(num))); i++ {
		if num%i == 0 {
			return
		}
	}

	totalPrimeNumbers++
}

func main() {

	startTime := time.Now()

	for i := 3; i <= MAX_INT; i++ {
		checkPrime(i)
	}

	// Doing [totalPrimeNumbers+1] to include 2(the only even prime) which we did not account for
	fmt.Println("Checking till", MAX_INT, "found", totalPrimeNumbers+1, "prime numbers. Time taken :", time.Since(startTime))

}
