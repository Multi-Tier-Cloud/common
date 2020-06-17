/* Copyright 2020 Multi-Tier-Cloud Development Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"fmt"
	"time"
)

type ExpoBackoff struct {
	initPeriod time.Duration
	maxPeriod  time.Duration
	nextPeriod time.Duration
}

// Sleeps for some duration, where each invocation of this method
// will exponentially increasing the duration.
func (eb *ExpoBackoff) Sleep() {
	eb.nextPeriod *= 2
	if eb.nextPeriod < eb.initPeriod {
		eb.nextPeriod = eb.initPeriod
	} else if eb.nextPeriod > eb.maxPeriod {
		eb.nextPeriod = eb.maxPeriod
	}
	time.Sleep(eb.nextPeriod)
}

// Creates a new ExpoBackoff.
// Parameters 'init' and 'max' denote initial and max durations to sleep
// whenever the Sleep() method is called.
func NewExpoBackoff(init, max time.Duration) (*ExpoBackoff, error) {
	if max < init {
		return nil, fmt.Errorf("Max duration cannot be less than init duration\n")
	}

	if init < 0 || max < 0 {
		return nil, fmt.Errorf("Init and max durations cannot be negative\n")
	}

	return &ExpoBackoff{
		initPeriod: init,
		maxPeriod:  max,
	}, nil
}

type ExpoBackoffAttempts struct {
	backoff     *ExpoBackoff
	maxAttempts int
	attempt     int
}

// Wrapper function around ExponentialBackoff's Sleep().
// Allows users to write for loops such as:
//  eba := NewExpoBackoffAttempts(1 * time.Second, 8 * time.Second, 6)
//  for eba.Attempt() {
//      // DO SOMETHING
//  }
//
// The above loop will execute 6 times, and sleep 5 times (1, 2, 4, 8, and
// 8 seconds) between loops. Note that the first call will not sleep.
//
// Returns
//  - False: If max attempts has been reached
//  - True: If max attempts has not been reached, will return true after
//          potentially sleeping. Note that the first call will not sleep.
func (eba *ExpoBackoffAttempts) Attempt() bool {
	if eba.attempt >= eba.maxAttempts {
		return false
	} else if eba.attempt == 0 {
		eba.attempt += 1
		return true
	} else {
		eba.attempt += 1
		eba.backoff.Sleep()
		return true
	}
}

// Creates a new ExpoBackoffAttempts
// Similar to ExpoBackoff, but limited in the number of times it can sleep
// See example usage in comments for the Attempt() method
func NewExpoBackoffAttempts(init, max time.Duration,
	attempts int) (*ExpoBackoffAttempts, error) {

	backoff, err := NewExpoBackoff(init, max)
	if err != nil {
		return nil, err
	}

	if attempts < 1 {
		return nil, fmt.Errorf("Must allow at least one attempt\n")
	}

	return &ExpoBackoffAttempts{
		backoff:     backoff,
		maxAttempts: attempts,
	}, nil
}
