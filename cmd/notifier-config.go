/*
 * Minio Cloud Storage, (C) 2017 Minio, Inc.
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

package cmd

import "sync"

// Notifier represents collection of supported notification queues.
type notifier struct {
	sync.RWMutex
	AMQP          amqpConfigs          `json:"amqp"`
	NATS          natsConfigs          `json:"nats"`
	ElasticSearch elasticSearchConfigs `json:"elasticsearch"`
	Redis         redisConfigs         `json:"redis"`
	PostgreSQL    postgreSQLConfigs    `json:"postgresql"`
	Kafka         kafkaConfigs         `json:"kafka"`
	Webhook       webhookConfigs       `json:"webhook"`
	// Add new notification queues.
}

type amqpConfigs map[string]amqpNotify

func (a amqpConfigs) Clone() amqpConfigs {
	a2 := make(amqpConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

type natsConfigs map[string]natsNotify

func (a natsConfigs) Clone() natsConfigs {
	a2 := make(natsConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

type elasticSearchConfigs map[string]elasticSearchNotify

func (a elasticSearchConfigs) Clone() elasticSearchConfigs {
	a2 := make(elasticSearchConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

type redisConfigs map[string]redisNotify

func (a redisConfigs) Clone() redisConfigs {
	a2 := make(redisConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

type postgreSQLConfigs map[string]postgreSQLNotify

func (a postgreSQLConfigs) Clone() postgreSQLConfigs {
	a2 := make(postgreSQLConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

type kafkaConfigs map[string]kafkaNotify

func (a kafkaConfigs) Clone() kafkaConfigs {
	a2 := make(kafkaConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

type webhookConfigs map[string]webhookNotify

func (a webhookConfigs) Clone() webhookConfigs {
	a2 := make(webhookConfigs, len(a))
	for k, v := range a {
		a2[k] = v
	}
	return a2
}

func (n *notifier) SetAMQPByID(accountID string, amqpn amqpNotify) {
	n.Lock()
	defer n.Unlock()
	n.AMQP[accountID] = amqpn
}

func (n *notifier) GetAMQP() map[string]amqpNotify {
	n.RLock()
	defer n.RUnlock()
	return n.AMQP.Clone()
}

func (n *notifier) GetAMQPByID(accountID string) amqpNotify {
	n.RLock()
	defer n.RUnlock()
	return n.AMQP[accountID]
}

func (n *notifier) SetNATSByID(accountID string, natsn natsNotify) {
	n.Lock()
	defer n.Unlock()
	n.NATS[accountID] = natsn
}

func (n *notifier) GetNATS() map[string]natsNotify {
	n.RLock()
	defer n.RUnlock()
	return n.NATS.Clone()
}

func (n *notifier) GetNATSByID(accountID string) natsNotify {
	n.RLock()
	defer n.RUnlock()
	return n.NATS[accountID]
}

func (n *notifier) SetElasticSearchByID(accountID string, es elasticSearchNotify) {
	n.Lock()
	defer n.Unlock()
	n.ElasticSearch[accountID] = es
}

func (n *notifier) GetElasticSearchByID(accountID string) elasticSearchNotify {
	n.RLock()
	defer n.RUnlock()
	return n.ElasticSearch[accountID]
}

func (n *notifier) GetElasticSearch() map[string]elasticSearchNotify {
	n.RLock()
	defer n.RUnlock()
	return n.ElasticSearch.Clone()
}

func (n *notifier) SetRedisByID(accountID string, r redisNotify) {
	n.Lock()
	defer n.Unlock()
	n.Redis[accountID] = r
}

func (n *notifier) GetRedis() map[string]redisNotify {
	n.RLock()
	defer n.RUnlock()
	return n.Redis.Clone()
}

func (n *notifier) GetRedisByID(accountID string) redisNotify {
	n.RLock()
	defer n.RUnlock()
	return n.Redis[accountID]
}

func (n *notifier) GetWebhook() map[string]webhookNotify {
	n.RLock()
	defer n.RUnlock()
	return n.Webhook.Clone()
}

func (n *notifier) GetWebhookByID(accountID string) webhookNotify {
	n.RLock()
	defer n.RUnlock()
	return n.Webhook[accountID]
}

func (n *notifier) SetWebhookByID(accountID string, pgn webhookNotify) {
	n.Lock()
	defer n.Unlock()
	n.Webhook[accountID] = pgn
}

func (n *notifier) SetPostgreSQLByID(accountID string, pgn postgreSQLNotify) {
	n.Lock()
	defer n.Unlock()
	n.PostgreSQL[accountID] = pgn
}

func (n *notifier) GetPostgreSQL() map[string]postgreSQLNotify {
	n.RLock()
	defer n.RUnlock()
	return n.PostgreSQL.Clone()
}

func (n *notifier) GetPostgreSQLByID(accountID string) postgreSQLNotify {
	n.RLock()
	defer n.RUnlock()
	return n.PostgreSQL[accountID]
}

func (n *notifier) SetKafkaByID(accountID string, kn kafkaNotify) {
	n.Lock()
	defer n.Unlock()
	n.Kafka[accountID] = kn
}

func (n *notifier) GetKafka() map[string]kafkaNotify {
	n.RLock()
	defer n.RUnlock()
	return n.Kafka.Clone()
}

func (n *notifier) GetKafkaByID(accountID string) kafkaNotify {
	n.RLock()
	defer n.RUnlock()
	return n.Kafka[accountID]
}
