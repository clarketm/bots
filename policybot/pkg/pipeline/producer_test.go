// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Take in a pr number from blob storage and examines the pr
// for all tests that are run and their results. The results are then written to storage.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestSetupError(t *testing.T) {
	expectedError := errors.New("fake error")
	sp := &IterProducer{
		Setup: func() error {
			return expectedError
		},
		Iterator: nil,
	}
	resultChan := sp.Start(context.TODO(), 1)
	result, ok := <-resultChan
	assert.Assert(t, ok)
	assert.ErrorType(t, result.Err(), expectedError)
	_, ok = <-resultChan
	assert.Assert(t, !ok)
}

func TestFake(t *testing.T) {
	var a []string
	b := []string{"foo"}
	c := append(a, b...)
	assert.Assert(t, c != nil)
}

func TestTransform(t *testing.T) {
	var things []interface{}
	for i := 0; i < 20; i++ {
		things = append(things, fmt.Sprintf("pathtopr/%d/somethingelse", i))
	}
	slt := StringLogTransformer{ErrHandler: func(e error) {
		t.Log(e)
	}}
	ctx := context.Background()
	sourceChan := BuildProducer(ctx, things)
	// our sample transform function returns only the part of the prpath that represents the pr number, and
	// only if the prnum is between high and low, inclusive
	resultChan := slt.Transform(ctx, sourceChan, func(prPath interface{}) (prNum interface{}, err error) {
		const high = 10
		const low = 7
		prParts := strings.Split(prPath.(string), "/")
		if len(prParts) < 2 {
			err = errors.New("too few segments in pr path")
			return
		}
		prNumInt, err := strconv.Atoi(prParts[len(prParts)-2])
		if err != nil {
			return
		} else if prNumInt <= high && prNumInt >= low {
			prNum = prParts[len(prParts)-2]
			return
		}
		err = ErrSkip
		return
	})
	for element := range resultChan {
		// no errors occur
		assert.NilError(t, element.Err())
		// zero is too low
		assert.Assert(t, element.Output() != "0")
		// eleven is too high
		assert.Assert(t, element.Output() != "11")
	}
}

func TestSuccess(t *testing.T) {
	things := []interface{}{"1", "2", "3", "4"}
	resultChan := BuildProducer(context.Background(), things)
	var resultCount int
	for result := range resultChan {
		assert.NilError(t, result.Err())
		resultCount++
	}
	assert.Equal(t, resultCount, len(things))
}
