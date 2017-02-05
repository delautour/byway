package bywayConfig

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/amerdrix/byway/core"
	"gopkg.in/redis.v5"
)

var redisClientSingleton *redis.Client

func withRedis(cb func(redis *redis.Client) error) error {

	if redisClientSingleton != nil {
		return cb(redisClientSingleton)
	}

	redisClientSingleton = redis.NewClient(&redis.Options{
		Addr:     "rawddss101v:6379",
		Password: "", // no password set
		DB:       9,  // use default DB
	})

	pong, err := redisClientSingleton.Ping().Result()
	if err != nil {
		log.Fatalf("byway: redis: %s\n", err)
	}
	log.Printf("byway: redis: %s", pong)

	return cb(redisClientSingleton)
}

func readRedisConfig(redis *redis.Client) *core.Config {
	config := core.NewConfig()

	rewriteMembers := redis.LRange("byway.rewrite", 0, -1)
	if rewriteMembers.Err() != nil {
		log.Fatalf("byway: redis: %s", rewriteMembers.Err())
	}
	log.Printf("byway: redis: %s\n", rewriteMembers)

	for _, rewritePattern := range rewriteMembers.Val() {
		config.Rewrites = append(config.Rewrites, core.RewriteConfigString(rewritePattern))
	}

	log.Printf("byway: rewrites: %s", config.Rewrites)

	indexName := "byway.service_index"
	indexMembers := redis.SMembers(indexName)
	if indexMembers.Err() != nil {
		log.Fatalf("byway: redis: %s", indexMembers.Err())
	}
	log.Printf("byway: redis: %s\n", indexMembers)

	for _, serviceName := range indexMembers.Val() {
		versionTable := make(map[core.VersionString]core.EndpointConfig)

		vtable := redis.HGetAll("byway.service." + serviceName)
		if vtable.Err() != nil {
			log.Fatalf("byway: redis: %s", vtable.Err())
		}
		log.Printf("byway: redis: %s\n", vtable)

		for serviceVersion, endpoint := range vtable.Val() {
			log.Printf("byway: redis: %s:%s \n", serviceVersion, endpoint)

			ep := core.EndpointConfig{}

			err := json.Unmarshal([]byte(endpoint), &ep)
			if err != nil {
				log.Fatalf("byway: redis: %s", err)
			}

			versionTable[core.VersionString(serviceVersion)] = ep
		}

		config.Mapping[core.ServiceName(serviceName)] = versionTable
	}

	for _, key := range redis.Keys("byway.topology.*").Val() {
		redisHash := redis.HGetAll(key)
		if redisHash.Err() != nil {
			log.Fatalf("byway: redis: %s", redisHash.Err())
		}

		vTable := make(map[core.ServiceName]core.VersionString)
		for service, version := range redisHash.Val() {
			vTable[core.ServiceName(service)] = core.VersionString(version)
		}

		config.Topologies[core.TopologyKey(key[15:len(key)])] = vTable

		log.Printf("byway: redis: topology: %s, %s", key, vTable)

	}

	return config
}

// CreateRewrite creates a rewrite rule
func CreateRewrite(rewrite core.RewriteConfigString) error {

	return withRedis(func(r *redis.Client) error {
		err := r.RPush("byway.rewrite", string(rewrite)).Err()
		if err != nil {
			return err
		}

		return r.Publish("byway.update", "go").Err()
	})
}

// CreateService creates an empty service
func CreateService(seviceName core.ServiceName) error {

	return withRedis(func(r *redis.Client) error {
		err := r.SAdd("byway.service_index", string(seviceName)).Err()
		if err != nil {
			return err
		}

		return r.Publish("byway.update", "go").Err()
	})
}

// CreateBinding creates binding to an endpoint
func CreateBinding(seviceName core.ServiceName, version core.VersionString, endpoint *core.EndpointConfig) error {
	return withRedis(func(r *redis.Client) error {
		err := r.SAdd("byway.service_index", string(seviceName)).Err()
		if err != nil {
			return err
		}

		config, err := json.Marshal(endpoint)

		err = r.HSet("byway.service."+string(seviceName), string(version), string(config)).Err()
		if err != nil {
			return err
		}

		return r.Publish("byway.update", "go").Err()
	})
}

// AddServiceToTopology adds a service binding to a topology
func AddServiceToTopology(key core.TopologyKey, service core.ServiceName, version core.VersionString) error {
	return withRedis(func(r *redis.Client) error {
		err := r.HSet("byway.topology."+string(key), string(service), string(version)).Err()
		if err != nil {
			return err
		}
		return r.Publish("byway.update", "go").Err()
	})
}

// RemoveRewrite creates a rewrite rule
func RemoveRewrite(index int64, rewrite core.RewriteConfigString) error {

	return withRedis(func(r *redis.Client) error {
		get := r.LRange("byway.rewrite", index, -1)
		if get.Err() != nil {
			return get.Err()
		}
		val := get.Val()

		if val == nil || len(val) == 0 || val[0] != string(rewrite) {
			return fmt.Errorf("Rewrite (%s) rule at index (%d) does not match provided rule", val, index)
		}
		err := r.LSet("byway.rewrite", index, ":DEL:").Err()
		if err != nil {
			return err
		}
		err = r.LRem("byway.rewrite", 0, ":DEL:").Err()
		if err != nil {
			return err
		}

		return r.Publish("byway.update", "go").Err()

	})
}

// WatchRedis - reads config from redis into the provided channel
func WatchRedis(channel chan *core.Config, exit chan bool) {
	withRedis(func(redis *redis.Client) error {
		subscription, err := redis.Subscribe("byway.update")

		if err != nil {
			log.Fatalf("byway: redis: %s\n", err)
		}

		go func() {
			for {
				subscription.Receive()
				channel <- readRedisConfig(redis)
			}
		}()
		return err
	})

}
