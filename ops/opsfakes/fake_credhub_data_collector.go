// Code generated by counterfeiter. DO NOT EDIT.
package opsfakes

import (
	"sync"

	"github.com/pivotal-cf/aqueduct-courier/credhub"
)

type FakeCredhubDataCollector struct {
	CollectStub        func() (credhub.Data, error)
	collectMutex       sync.RWMutex
	collectArgsForCall []struct{}
	collectReturns     struct {
		result1 credhub.Data
		result2 error
	}
	collectReturnsOnCall map[int]struct {
		result1 credhub.Data
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCredhubDataCollector) Collect() (credhub.Data, error) {
	fake.collectMutex.Lock()
	ret, specificReturn := fake.collectReturnsOnCall[len(fake.collectArgsForCall)]
	fake.collectArgsForCall = append(fake.collectArgsForCall, struct{}{})
	fake.recordInvocation("Collect", []interface{}{})
	fake.collectMutex.Unlock()
	if fake.CollectStub != nil {
		return fake.CollectStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.collectReturns.result1, fake.collectReturns.result2
}

func (fake *FakeCredhubDataCollector) CollectCallCount() int {
	fake.collectMutex.RLock()
	defer fake.collectMutex.RUnlock()
	return len(fake.collectArgsForCall)
}

func (fake *FakeCredhubDataCollector) CollectReturns(result1 credhub.Data, result2 error) {
	fake.CollectStub = nil
	fake.collectReturns = struct {
		result1 credhub.Data
		result2 error
	}{result1, result2}
}

func (fake *FakeCredhubDataCollector) CollectReturnsOnCall(i int, result1 credhub.Data, result2 error) {
	fake.CollectStub = nil
	if fake.collectReturnsOnCall == nil {
		fake.collectReturnsOnCall = make(map[int]struct {
			result1 credhub.Data
			result2 error
		})
	}
	fake.collectReturnsOnCall[i] = struct {
		result1 credhub.Data
		result2 error
	}{result1, result2}
}

func (fake *FakeCredhubDataCollector) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.collectMutex.RLock()
	defer fake.collectMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeCredhubDataCollector) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
