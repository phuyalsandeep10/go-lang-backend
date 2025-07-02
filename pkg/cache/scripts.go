package cache

import (
	"github.com/go-redis/redis/v8"
)

// Lua scripts for Redis operations
var (
	setSearchResultScript        *redis.Script
	invalidatePropertyCacheScript *redis.Script
)

func init() {
	// store search results and associates the search key with property IDs.
	setSearchResultScript = redis.NewScript(`
		local search_key = ARGV[1]
		local property_ids_json = ARGV[2]
		local search_expiration = tonumber(ARGV[3])
		redis.call('SET', search_key, property_ids_json)
		redis.call('EXPIRE', search_key, search_expiration)
		for i = 4, #ARGV do
			local property_id = ARGV[i]
			local set_key = 'property:keys:' .. property_id
			redis.call('SADD', set_key, search_key)
			redis.call('EXPIRE', set_key, 3600)
		end
		return 1
	`)

	// remove all cache keys associated with a property.
	invalidatePropertyCacheScript = redis.NewScript(`
		local set_key = 'property:keys:' .. ARGV[1]
		local cache_keys = redis.call('SMEMBERS', set_key)
		if #cache_keys > 0 then
			redis.call('DEL', unpack(cache_keys))
		end
		redis.call('DEL', set_key)
		return 1
	`)
}
