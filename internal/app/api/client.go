// Copyright 2018 Clivern. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"fmt"
	"github.com/clivern/beaver/internal/app/driver"
	"github.com/clivern/beaver/internal/pkg/logger"
	"github.com/clivern/beaver/internal/pkg/utils"
	"os"
	"time"
)

// ClientsHashPrefix is the hash prefix
const ClientsHashPrefix string = "beaver.client"

// Client struct
type Client struct {
	Driver        *driver.Redis
	CorrelationID string
}

// ClientResult struct
type ClientResult struct {
	ID        string   `json:"id"`
	Token     string   `json:"token"`
	Channels  []string `json:"channels"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// LoadFromJSON load object from json
func (c *ClientResult) LoadFromJSON(data []byte) (bool, error) {
	err := json.Unmarshal(data, &c)
	if err != nil {
		return false, err
	}
	return true, nil
}

// ConvertToJSON converts object to json
func (c *ClientResult) ConvertToJSON() (string, error) {
	data, err := json.Marshal(&c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateClient generates client ID & Token
func (c *ClientResult) GenerateClient() (bool, error) {

	now := time.Now().Unix()
	c.ID = utils.GenerateUUID4()

	token, err := utils.GenerateJWTToken(
		fmt.Sprintf("%s@%d", c.ID, now),
		now,
		os.Getenv("AppSecret"),
	)

	if err != nil {
		return false, err
	}

	c.Token = token
	c.CreatedAt = now
	c.UpdatedAt = now

	return true, nil
}

// Init initialize the redis connection
func (c *Client) Init() bool {
	c.Driver = driver.NewRedisDriver()

	result, err := c.Driver.Connect()
	if !result {
		logger.Errorf(
			`Error while connecting to redis: %s {"correlationId":"%s"}`,
			err.Error(),
			c.CorrelationID,
		)
		return false
	}
	return true
}

// CreateClient creates a client
func (c *Client) CreateClient(client ClientResult) (bool, error) {

	exists, err := c.Driver.HExists(ClientsHashPrefix, client.ID)

	if err != nil {
		logger.Errorf(
			`Error while creating client %s: %s {"correlationId":"%s"}`,
			client.ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while creating client %s`,
			client.ID,
		)
	}

	if exists {
		logger.Warningf(
			`Trying to create existent client %s {"correlationId":"%s"}`,
			client.ID,
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Trying to create existent client %s`,
			client.ID,
		)
	}

	result, err := client.ConvertToJSON()

	if err != nil {
		logger.Errorf(
			`Something wrong with client %s data: %s {"correlationId":"%s"}`,
			client.ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Something wrong with client %s data`,
			client.ID,
		)
	}

	_, err = c.Driver.HSet(ClientsHashPrefix, client.ID, result)

	if err != nil {
		logger.Errorf(
			`Error while creating client %s: %s {"correlationId":"%s"}`,
			client.ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while creating client %s`,
			client.ID,
		)
	}

	for _, channel := range client.Channels {
		ok, err := c.AddToChannel(client.ID, channel)
		if !ok || err != nil {
			return false, err
		}
	}

	return true, nil
}

// GetClientByID gets a client by ID
func (c *Client) GetClientByID(ID string) (ClientResult, error) {

	var clientResult ClientResult

	exists, err := c.Driver.HExists(ClientsHashPrefix, ID)

	if err != nil {
		logger.Errorf(`Error while getting client %s: %s {"correlationId":"%s"}`, ID, err.Error(), c.CorrelationID)
		return clientResult, fmt.Errorf("Error while getting client %s", ID)
	}

	if !exists {
		logger.Warningf(`Trying to get non existent client %s {"correlationId":"%s"}`, ID, c.CorrelationID)
		return clientResult, fmt.Errorf("Trying to get non existent client %s", ID)
	}

	value, err := c.Driver.HGet(ClientsHashPrefix, ID)

	if err != nil {
		logger.Errorf(`Error while getting client %s: %s {"correlationId":"%s"}`, ID, err.Error(), c.CorrelationID)
		return clientResult, fmt.Errorf("Error while getting client %s", ID)
	}

	_, err = clientResult.LoadFromJSON([]byte(value))

	if err != nil {
		logger.Errorf(`Error while getting client %s: %s {"correlationId":"%s"}`, ID, err.Error(), c.CorrelationID)
		return clientResult, fmt.Errorf("Error while getting client %s", ID)
	}

	return clientResult, nil
}

// UpdateClientByID updates a client by ID
func (c *Client) UpdateClientByID(client ClientResult) (bool, error) {

	exists, err := c.Driver.HExists(ClientsHashPrefix, client.ID)

	if err != nil {
		logger.Errorf(
			`Error while updating client %s: %s {"correlationId":"%s"}`,
			client.ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while updating client %s`,
			client.ID,
		)
	}

	if !exists {
		logger.Warningf(
			`Trying to create non existent client %s {"correlationId":"%s"}`,
			client.ID,
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Trying to create non existent client %s`,
			client.ID,
		)
	}

	result, err := client.ConvertToJSON()

	if err != nil {
		logger.Errorf(
			`Something wrong with client %s data: %s {"correlationId":"%s"}`,
			client.ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Something wrong with client %s data`,
			client.ID,
		)
	}

	_, err = c.Driver.HSet(ClientsHashPrefix, client.ID, result)

	if err != nil {
		logger.Errorf(
			`Error while updating client %s: %s {"correlationId":"%s"}`,
			client.ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while updating client %s`,
			client.ID,
		)
	}

	return true, nil
}

// DeleteClientByID deletes a client with ID
func (c *Client) DeleteClientByID(ID string) (bool, error) {

	client, err := c.GetClientByID(ID)

	if err != nil {
		logger.Errorf(
			`Error while deleting client %s: %s {"correlationId":"%s"}`,
			ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while deleting client %s`,
			ID,
		)
	}

	for _, channel := range client.Channels {
		ok, err := c.RemoveFromChannel(ID, channel)
		if !ok || err != nil {
			return false, err
		}
	}

	// Remove client from clients
	_, err = c.Driver.HDel(ClientsHashPrefix, ID)

	if err != nil {
		logger.Errorf(
			`Error while deleting client %s: %s {"correlationId":"%s"}`,
			ID,
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while deleting client %s`,
			ID,
		)
	}

	return true, nil
}

// Unsubscribe from channels
func (c *Client) Unsubscribe(ID string, channels []string) (bool, error) {

	validator := utils.Validator{}
	clientResult, err := c.GetClientByID(ID)

	if err != nil {
		return false, err
	}

	for i, channel := range channels {
		if validator.IsIn(channel, clientResult.Channels) {
			ok, err := c.RemoveFromChannel(ID, channel)
			if !ok || err != nil {
				return false, err
			}
			clientResult.Channels = utils.Unset(clientResult.Channels, i)
		}
	}

	return c.UpdateClientByID(clientResult)
}

// Subscribe to channels
func (c *Client) Subscribe(ID string, channels []string) (bool, error) {

	validator := utils.Validator{}
	clientResult, err := c.GetClientByID(ID)

	if err != nil {
		return false, err
	}

	for _, channel := range channels {
		if !validator.IsIn(channel, clientResult.Channels) {
			ok, err := c.AddToChannel(ID, channel)
			if !ok || err != nil {
				return false, err
			}
			clientResult.Channels = append(clientResult.Channels, channel)
		}
	}

	return c.UpdateClientByID(clientResult)
}

// AddToChannel adds a client to a channel
func (c *Client) AddToChannel(ID string, channel string) (bool, error) {
	// Remove client from channel subscribers
	_, err := c.Driver.HSet(fmt.Sprintf("%s.subscribers", channel), ID, "")

	if err != nil {
		logger.Errorf(
			`Error while adding client %s to channel %s: %s {"correlationId":"%s"}`,
			ID,
			fmt.Sprintf("%s.subscribers", channel),
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while adding client %s to channel %s`,
			ID,
			fmt.Sprintf("%s.subscribers", channel),
		)
	}

	return true, nil
}

// RemoveFromChannel removes a client from a channel
func (c *Client) RemoveFromChannel(ID string, channel string) (bool, error) {
	// Remove client from channel subscribers
	_, err := c.Driver.HDel(fmt.Sprintf("%s.subscribers", channel), ID)

	if err != nil {
		logger.Errorf(
			`Error while removing client %s from channel %s: %s {"correlationId":"%s"}`,
			ID,
			fmt.Sprintf("%s.subscribers", channel),
			err.Error(),
			c.CorrelationID,
		)
		return false, fmt.Errorf(
			`Error while removing client %s from %s`,
			ID,
			fmt.Sprintf("%s.subscribers", channel),
		)
	}

	return true, nil
}

// Connect a client
func (c *Client) Connect(clientResult ClientResult) (bool, error) {
	for _, channel := range clientResult.Channels {
		// Remove client from channel listeners
		_, err := c.Driver.HSet(fmt.Sprintf("%s.listeners", channel), clientResult.ID, "")

		if err != nil {
			logger.Errorf(
				`Error while adding client %s to channel %s: %s {"correlationId":"%s"}`,
				clientResult.ID,
				fmt.Sprintf("%s.listeners", channel),
				err.Error(),
				c.CorrelationID,
			)
			return false, fmt.Errorf(
				`Error while adding client %s to channel %s`,
				clientResult.ID,
				fmt.Sprintf("%s.listeners", channel),
			)
		}
	}

	return true, nil
}

// Disconnect a client
func (c *Client) Disconnect(clientResult ClientResult) (bool, error) {
	for _, channel := range clientResult.Channels {
		// Remove client from channel listeners
		_, err := c.Driver.HDel(fmt.Sprintf("%s.listeners", channel), clientResult.ID)

		if err != nil {
			logger.Errorf(
				`Error while removing client %s from channel %s: %s {"correlationId":"%s"}`,
				clientResult.ID,
				fmt.Sprintf("%s.listeners", channel),
				err.Error(),
				c.CorrelationID,
			)
			return false, fmt.Errorf(
				"Error while removing client %s from %s",
				clientResult.ID,
				fmt.Sprintf("%s.listeners", channel),
			)
		}
	}

	return true, nil
}
