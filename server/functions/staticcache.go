package main

import "sync"

var busStaticMapCache sync.Map

func cachedBusStaticMap(prefix string) ([]busStationmap, bool) {
	v, ok := busStaticMapCache.Load(prefix)
	if !ok {
		return nil, false
	}
	list, ok := v.([]busStationmap)
	if !ok {
		return nil, false
	}
	return list, true
}

func storeBusStaticMap(prefix string, list []busStationmap) {
	busStaticMapCache.Store(prefix, list)
}

func invalidateBusStaticMap() {
	busStaticMapCache.Range(func(key, _ any) bool {
		busStaticMapCache.Delete(key)
		return true
	})
}
