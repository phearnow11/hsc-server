// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// CloudWorkloadSecurityAgentRuleCreateData Object for a single Agent rule.
type CloudWorkloadSecurityAgentRuleCreateData struct {
	// Create a new Cloud Workload Security Agent rule.
	Attributes CloudWorkloadSecurityAgentRuleCreateAttributes `json:"attributes"`
	// The type of the resource. The value should always be `agent_rule`.
	Type CloudWorkloadSecurityAgentRuleType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCloudWorkloadSecurityAgentRuleCreateData instantiates a new CloudWorkloadSecurityAgentRuleCreateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudWorkloadSecurityAgentRuleCreateData(attributes CloudWorkloadSecurityAgentRuleCreateAttributes, typeVar CloudWorkloadSecurityAgentRuleType) *CloudWorkloadSecurityAgentRuleCreateData {
	this := CloudWorkloadSecurityAgentRuleCreateData{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewCloudWorkloadSecurityAgentRuleCreateDataWithDefaults instantiates a new CloudWorkloadSecurityAgentRuleCreateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudWorkloadSecurityAgentRuleCreateDataWithDefaults() *CloudWorkloadSecurityAgentRuleCreateData {
	this := CloudWorkloadSecurityAgentRuleCreateData{}
	var typeVar CloudWorkloadSecurityAgentRuleType = CLOUDWORKLOADSECURITYAGENTRULETYPE_AGENT_RULE
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *CloudWorkloadSecurityAgentRuleCreateData) GetAttributes() CloudWorkloadSecurityAgentRuleCreateAttributes {
	if o == nil {
		var ret CloudWorkloadSecurityAgentRuleCreateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateData) GetAttributesOk() (*CloudWorkloadSecurityAgentRuleCreateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *CloudWorkloadSecurityAgentRuleCreateData) SetAttributes(v CloudWorkloadSecurityAgentRuleCreateAttributes) {
	o.Attributes = v
}

// GetType returns the Type field value.
func (o *CloudWorkloadSecurityAgentRuleCreateData) GetType() CloudWorkloadSecurityAgentRuleType {
	if o == nil {
		var ret CloudWorkloadSecurityAgentRuleType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateData) GetTypeOk() (*CloudWorkloadSecurityAgentRuleType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *CloudWorkloadSecurityAgentRuleCreateData) SetType(v CloudWorkloadSecurityAgentRuleType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudWorkloadSecurityAgentRuleCreateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudWorkloadSecurityAgentRuleCreateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *CloudWorkloadSecurityAgentRuleCreateAttributes `json:"attributes"`
		Type       *CloudWorkloadSecurityAgentRuleType             `json:"type"`
	}{}
	all := struct {
		Attributes CloudWorkloadSecurityAgentRuleCreateAttributes `json:"attributes"`
		Type       CloudWorkloadSecurityAgentRuleType             `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("Required field attributes missing")
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Type = all.Type
	return nil
}