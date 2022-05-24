package thing

import "sync"

// ThingStore is a simple in-memory store of Thing Description documents
type ThingStore struct {
	// Account whose TD's are held here
	accountID string

	// tdMap is a map of TD documents by Thing ID
	tdMap map[string]*ThingTD

	// tdMapMutex for safe concurrent access to the TD store
	tdMapMutex sync.RWMutex
}

// AddTD adds or replaces the store with the provided TD
func (ts *ThingStore) AddTD(td *ThingTD) {
	ts.Update(td)
}

// GetIDs returns the array of thing IDs
func (ts *ThingStore) GetIDs() []string {
	idList := make([]string, len(ts.tdMap))
	for key := range ts.tdMap {
		idList = append(idList, key)
	}
	return idList
}

// Load the thing store cached data (if any)
// TODO: not implemented
func (ts *ThingStore) Load() {
	// todo
}

// GetByID returns the TD of the Thing with the given id
func (ts *ThingStore) GetByID(thingID string) *ThingTD {
	ts.tdMapMutex.RLock()
	defer ts.tdMapMutex.RUnlock()
	td := ts.tdMap[thingID]
	return td
}

// Save the thing store cached data (if any)
// TODO: not implemented
func (ts *ThingStore) Save() {
	// todo
}

// Update adds or replaces a new discovered ThingTD in the collection
// This will do some cleanup on the TD to ensure that properties, actions, and events
// include their own name.
// @param td with the TD to update. This can be modified
func (ts *ThingStore) Update(td *ThingTD) {
	ts.tdMapMutex.Lock()
	defer ts.tdMapMutex.Unlock()

	ts.tdMap[td.ID] = td

	// augment the properties, events and actions with their name for ease of use
	//if td2.Properties != nil {
	//	for key, val := range td2.Properties {
	//		val.DisplayName = key
	//	}
	//}
	//if td2.Actions != nil {
	//	for key, val := range td.Actions {
	//		val.DisplayName = key
	//	}
	//}
	//if td.Events != nil {
	//	for key, val := range td.Events {
	//		val.DisplayName = key
	//	}
	//}
}

// NewThingStore creates a new instance of the TD store for the given account
func NewThingStore(accountID string) *ThingStore {
	ts := &ThingStore{
		accountID:  accountID,
		tdMap:      make(map[string]*ThingTD),
		tdMapMutex: sync.RWMutex{},
	}
	return ts
}
