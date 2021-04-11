package kafka

import (
	"fmt"
	"reflect"

	"github.com/Shopify/sarama"
	"github.com/mitchellh/mapstructure"
)

func stringToBalanceStrategyHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*sarama.BalanceStrategy)(nil)).Elem() {
			return data, nil
		}

		strategyStr := data.(string)
		var strategy sarama.BalanceStrategy

		switch strategyStr {
		case "sticky":
			strategy = sarama.BalanceStrategySticky
		case "roundrobin":
			strategy = sarama.BalanceStrategyRoundRobin
		case "range":
			strategy = sarama.BalanceStrategyRange
		default:
			return nil, fmt.Errorf("unrecognized balance strategy: %q", strategyStr)
		}

		return strategy, nil
	}
}

func stringToKafkaVersionHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf((*sarama.KafkaVersion)(nil)).Elem() {
			return data, nil
		}

		return sarama.ParseKafkaVersion(data.(string))
	}
}
