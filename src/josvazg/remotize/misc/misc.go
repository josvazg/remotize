// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// This package is a used by remotize and other subpackages but contains code
// it is uninteresting to the user or that it should normally NOT need to use. 
//
package misc

import (
	"fmt"
	"rpc"
	"reflect"
	"sync"
)

// Remotized Registry
var registry = make(map[string]interface{})

// Registry's lock
var lock sync.RWMutex

// BuildService returns a service wrapper for an interface on a given RpcServer.
//
// Users DON'T need to care about this, as it is done for them by the 
// autogenerated code and will be invoked as appropiate when calling NewService.
type BuildService func(*rpc.Server, interface{}) interface{}

// BuildRemote returns a local reference to a remote interface reachable through
// a given RpcClient.
//
// Users DON'T need to care about this, as it is done for them by the 
// autogenerated code and will be invoked as appropiate when calling NewRemote.
type BuildRemote func(*rpc.Client) interface{}

// Register will record a local reference to a remote interface 'r', 
// its builder 'br', the corresponding service 's' and its builder, so that they
// can be retrieved later by NewRemote or NewService calls respectively.
//
// Users DON'T need to care about this registration, as it is done by the 
// autogenerated code for them.
func Register(r interface{}, br BuildRemote, s interface{}, bs BuildService) {
	cname := fmt.Sprintf("%v", reflect.TypeOf(r))
	sname := fmt.Sprintf("%v", reflect.TypeOf(s))
	lock.Lock()
	defer lock.Unlock()
	registry[cname] = br
	registry[sname] = bs
}

// registryFind will find a registered name in the remotize registry
func RegistryFind(name string) interface{} {
	lock.Lock()
	defer lock.Unlock()
	return registry[name]
}

// Suffix will return the proper "r" or "er" or "" ending as an interface for 
// the given 'name'
func Suffix(name string) string {
	s := ""
	if !EndsWith(name, "er") {
		if EndsWithVowel(name) {
			s = "r"
		} else {
			s = "er"
		}
	}
	return s
}

// StartsWith returns true if str starts with substring s
func StartsWith(str, s string) bool {
	return len(str) >= len(s) && str[:len(s)] == s
}

// EndsWith returns true if str ends with substring s
func EndsWith(str, s string) bool {
	return len(str) >= len(s) && str[len(str)-len(s):] == s
}

// EndsWithVowel returns true if str ends with an ASCII vowel (a,e,i,o,u)
func EndsWithVowel(str string) bool {
	vowels := []string{"a", "e", "i", "o", "u"}
	for _, v := range vowels {
		if str[len(str)-len(v):] == v {
			return true
		}
	}
	return false
}

