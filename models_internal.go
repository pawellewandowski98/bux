package bux

import (
	"context"
	"time"

	"github.com/BuxOrg/bux/notifications"
)

// AfterDeleted will fire after a successful delete in the Datastore
func (m *Model) AfterDeleted(_ context.Context) error {
	m.DebugLog("starting: " + m.Name() + " AfterDelete hook...")
	m.DebugLog("end: " + m.Name() + " AfterDelete hook")
	return nil
}

// BeforeUpdating will fire before updating a model in the Datastore
func (m *Model) BeforeUpdating(_ context.Context) error {
	m.DebugLog("starting: " + m.Name() + " BeforeUpdate hook...")
	m.DebugLog("end: " + m.Name() + " BeforeUpdate hook")
	return nil
}

// Client will return the current client
func (m *Model) Client() ClientInterface {
	return m.client
}

// ChildModels will return any child models
func (m *Model) ChildModels() []ModelInterface {
	return nil
}

// DebugLog will display verbose logs
func (m *Model) DebugLog(text string) {
	c := m.Client()
	if c != nil && c.IsDebug() {
		c.Logger().Info(context.Background(), text)
	}
}

// enrich is run after getting a record from the database
func (m *Model) enrich(name ModelName, opts ...ModelOps) {
	// Set the name
	m.name = name

	// Overwrite defaults
	m.SetOptions(opts...)
}

// GetOptions will get the options that are set on that model
func (m *Model) GetOptions(isNewRecord bool) (opts []ModelOps) {

	// Client was set on the model
	if m.client != nil {
		opts = append(opts, WithClient(m.client))
	}

	// New record flag
	if isNewRecord {
		opts = append(opts, New())
	}

	return
}

// IsNew returns true if the model is (or was) a new record
func (m *Model) IsNew() bool {
	return m.newRecord
}

// GetID will get the model id, if overwritten in the actual model
func (m *Model) GetID() string {
	return ""
}

// Name will get the collection name (model)
func (m *Model) Name() string {
	return m.name.String()
}

// New will set the record to new
func (m *Model) New() {
	m.newRecord = true
}

// NotNew sets newRecord to false
func (m *Model) NotNew() {
	m.newRecord = false
}

// RawXpub returns the rawXpubKey
func (m *Model) RawXpub() string {
	return m.rawXpubKey
}

// SetRecordTime will set the record timestamps (created is true for a new record)
func (m *Model) SetRecordTime(created bool) {
	if created {
		m.CreatedAt = time.Now().UTC()
	} else {
		m.UpdatedAt = time.Now().UTC()
	}
}

// UpdateMetadata will update the metadata on the model
// any key set to nil will be removed, other keys updated or added
func (m *Model) UpdateMetadata(metadata Metadata) {
	if m.Metadata == nil {
		m.Metadata = make(Metadata)
	}

	for key, value := range metadata {
		if value == nil {
			delete(m.Metadata, key)
		} else {
			m.Metadata[key] = value
		}
	}
}

// SetOptions will set the options on the model
func (m *Model) SetOptions(opts ...ModelOps) {
	for _, opt := range opts {
		opt(m)
	}
}

// Display filter the model for display
func (m *Model) Display() interface{} {
	return m
}

// RegisterTasks will register the model specific tasks on client initialization
func (m *Model) RegisterTasks() error {
	return nil
}

// AfterUpdated will fire after a successful update into the Datastore
func (m *Model) AfterUpdated(_ context.Context) error {
	m.DebugLog("starting: " + m.Name() + " AfterUpdated hook...")
	m.DebugLog("end: " + m.Name() + " AfterUpdated hook")
	return nil
}

// AfterCreated will fire after the model is created in the Datastore
func (m *Model) AfterCreated(_ context.Context) error {
	m.DebugLog("starting: " + m.Name() + " AfterCreated hook...")
	m.DebugLog("end: " + m.Name() + " AfterCreated hook")
	return nil
}

// incrementField will increment the given field atomically in the datastore
func incrementField(ctx context.Context, model ModelInterface, fieldName string,
	increment int64) (int64, error) {

	// Debug log
	// model.DebugLog(fmt.Sprintf("increment model %s field ... %s %d", model.Name(), fieldName, increment))

	// Check for client
	c := model.Client()
	if c == nil {
		return 0, ErrMissingClient
	}

	// Increment
	newValue, err := c.Datastore().IncrementModel(ctx, model, fieldName, increment)
	if err != nil {
		return 0, err
	}

	// Fire an AfterUpdate event
	if err = model.AfterUpdated(ctx); err != nil {
		return 0, err
	}
	return newValue, nil
}

// notify about an event on the model
func notify(eventType notifications.EventType, model interface{}) {

	// run the notifications in a separate goroutine since there could be significant network delay
	// communicating with a notification provider

	go func() {
		m := model.(ModelInterface)
		if client := m.Client(); client != nil {
			if n := client.Notifications(); n != nil {
				if err := n.Notify(
					context.Background(), m.GetModelName(), eventType, model, m.GetID(),
				); err != nil {
					client.Logger().Error(
						context.Background(),
						"failed notifying about "+string(eventType)+" on "+m.GetID()+": "+err.Error(),
					)
				}
			}
		}
	}()
}
