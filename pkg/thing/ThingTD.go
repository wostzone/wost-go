package thing

import (
	"encoding/json"
	"sync"
	"time"

	grpcthing "github.com/wostzone/wost.grpc/go/thing"

	"github.com/wostzone/wost-go/pkg/vocab"
)

// ThingTD wraps the grpc Thing message struct and adds method to simplify creating TDs
type ThingTD struct {
	grpcthing.ThingDescription
	updateMutex sync.RWMutex
}

// AddAction provides a simple way to add an action affordance Schema to the TD
// This returns the action affordance that can be augmented/modified directly
//
// name is the name under which it is stored in the action affordance map. Any existing name will be replaced.
// title is the title used in the action. It is okay to use name if not sure.
// dataType is the type of data the action holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
func (tdoc *ThingTD) AddAction(name string, title string, dataType string) *grpcthing.ActionAffordance {
	actionAff := &grpcthing.ActionAffordance{
		Title: title,
		Input: &grpcthing.DataSchema{
			Title:    title,
			Type:     dataType,
			ReadOnly: true,
		},
	}
	tdoc.UpdateAction(name, actionAff)
	return actionAff
}

// AddProperty provides a simple way to add a property to the TD
// This returns the property affordance that can be augmented/modified directly
// By default the property is a read-only attribute.
//
// name is the name under which it is stored in the property affordance map. Any existing name will be replaced.
// title is the title used in the property. It is okay to use name if not sure.
// dataType is the type of data the property holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
func (tdoc *ThingTD) AddProperty(name string, title string, dataType string) *grpcthing.PropertyAffordance {
	prop := &grpcthing.PropertyAffordance{
		Title:    title,
		Type:     dataType,
		ReadOnly: true,
	}
	tdoc.UpdateProperty(name, prop)
	return prop
}

// AddEvent provides a simple way to add an event to the TD
// This returns the event affordance that can be augmented/modified directly
//
// name is the name under which it is stored in the property affordance map. Any existing name will be replaced.
// title is the title used in the event. It is okay to use name if not sure.
// dataType is the type of data the event holds, WoTDataTypeNumber, ..Object, ..Array, ..String, ..Integer, ..Boolean or null
func (tdoc *ThingTD) AddEvent(name string, title string, dataType string) *grpcthing.EventAffordance {
	evAff := &grpcthing.EventAffordance{
		Title: title,
		Data: &grpcthing.DataSchema{
			Title:    title,
			Type:     dataType,
			ReadOnly: true,
		},
	}
	tdoc.UpdateEvent(name, evAff)
	return evAff
}

// AsMap returns the TD document as a map
func (tdoc *ThingTD) AsMap() map[string]interface{} {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	var asMap map[string]interface{}
	asJSON, _ := json.Marshal(tdoc)
	json.Unmarshal(asJSON, &asMap)
	return asMap
}

// tbd json-ld parsers:
// Most popular; https://github.com/xeipuuv/gojsonschema
// Other:  https://github.com/piprate/json-gold

// GetAction returns the action affordance with Schema for the action.
// Returns nil if name is not an action or no affordance is defined.
func (tdoc *ThingTD) GetAction(name string) *grpcthing.ActionAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	actionAffordance, found := tdoc.Actions[name]
	if !found {
		return nil
	}
	return actionAffordance
}

// GetEvent returns the Schema for the event or nil if the event doesn't exist
func (tdoc *ThingTD) GetEvent(name string) *grpcthing.EventAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()

	eventAffordance, found := tdoc.Events[name]
	if !found {
		return nil
	}
	return eventAffordance
}

// GetProperty returns the Schema and value for the property or nil if name is not a property
func (tdoc *ThingTD) GetProperty(name string) *grpcthing.PropertyAffordance {
	tdoc.updateMutex.RLock()
	defer tdoc.updateMutex.RUnlock()
	propAffordance, found := tdoc.Properties[name]
	if !found {
		return nil
	}
	return propAffordance
}

// GetID returns the ID of the thing TD
func (tdoc *ThingTD) GetID() string {
	return tdoc.Id
}

// UpdateAction adds a new or replaces an existing action affordance (Schema) of name. Intended for creating TDs
// Use UpdateProperty if name is a property name.
// Returns the added affordance to support chaining
func (tdoc *ThingTD) UpdateAction(name string, affordance *grpcthing.ActionAffordance) *grpcthing.ActionAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Actions[name] = affordance
	return affordance
}

// UpdateEvent adds a new or replaces an existing event affordance (Schema) of name. Intended for creating TDs
// Returns the added affordance to support chaining
func (tdoc *ThingTD) UpdateEvent(name string, affordance *grpcthing.EventAffordance) *grpcthing.EventAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Events[name] = affordance
	return affordance
}

// UpdateForms sets the top level forms section of the TD
// NOTE: In WoST actions are always routed via the Hub using the Hub's protocol binding.
// Under normal circumstances forms are therefore not needed.
func (tdoc *ThingTD) UpdateForms(formList []*grpcthing.Form) {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Forms = formList
}

// UpdateProperty adds or replaces a property affordance in the TD. Intended for creating TDs
// Returns the added affordance to support chaining
func (tdoc *ThingTD) UpdateProperty(name string, affordance *grpcthing.PropertyAffordance) *grpcthing.PropertyAffordance {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Properties[name] = affordance
	return affordance
}

// UpdateTitleDescription sets the title and description of the Thing in the default language
func (tdoc *ThingTD) UpdateTitleDescription(title string, description string) {
	tdoc.updateMutex.Lock()
	defer tdoc.updateMutex.Unlock()
	tdoc.Title = title
	tdoc.Description = description
}

//// UpdateStatus sets the status property of a Thing
//// The status property is an object that holds possible status values
//// For example, an error status can be set using the 'error' field of the status property
//func (tdoc *ThingTD) UpdateStatus(statusName string, value string) {
//	sprop := tdoc.GetProperty("status")
//	if sprop == nil {
//		sprop = &PropertyAffordance{}
//		sprop.Title = "Status"
//		sprop.Description = "Device status info"
//		sprop.Type = vocab.WoTDataTypeObject
//	}
//	tdoc.UpdatePropertyValue("status", errorStatus)
//	// FIXME:is this a property
//	status := td["status"]
//	if status == nil {
//		status = make(map[string]interface{})
//		td["status"] = status
//	}
//	status.(map[string]interface{})["error"] = errorStatus
//}

// CreateTD creates a new Thing Description document with properties, events and actions
// Its structure:
// {
//      @context: "http://www.w3.org/ns/td",
//      id: <thingID>,      		// required in WoST. See CreateThingID for recommended format
//      title: string,              // required. Human description of the thing
//      @type: <deviceType>,        // required in WoST. See WoST DeviceType vocabulary
//      created: <iso8601>,         // will be the current timestamp. See vocabulary TimeFormat
//      actions: {name:TDAction, ...},
//      events:  {name: TDEvent, ...},
//      properties: {name: TDProperty, ...}
// }
func CreateTD(thingID string, title string, deviceType vocab.DeviceType) *ThingTD {
	td := ThingTD{
		ThingDescription: grpcthing.ThingDescription{
			AtContext:  []string{"http://www.w3.org/ns/thing"},
			Actions:    map[string]*grpcthing.ActionAffordance{},
			Created:    time.Now().Format(vocab.TimeFormat),
			Events:     map[string]*grpcthing.EventAffordance{},
			Forms:      nil,
			Id:         thingID,
			Modified:   time.Now().Format(vocab.TimeFormat),
			Properties: map[string]*grpcthing.PropertyAffordance{},
			// security schemas don't apply to WoST devices, except services exposed by the hub itself
			Security: []string{vocab.WoTNoSecurityScheme},
			Title:    title,
		},
		updateMutex: sync.RWMutex{},
	}

	// TODO @type is a JSON-LD keyword to label using semantic tags, eg it needs a Schema
	if deviceType != "" {
		// deviceType must be a string for serialization and querying
		td.AtType = []string{string(deviceType)}
	}
	return &td
}
