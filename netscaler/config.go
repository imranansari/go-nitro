/*
Copyright 2016 Citrix Systems, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package netscaler

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

//AddResource adds a resource of supplied type and name
func (c *NitroClient) AddResource(resourceType string, name string, resourceStruct interface{}) (string, error) {

	if c.ResourceExists(resourceType, name) == false {

		nsResource := make(map[string]interface{})
		nsResource[resourceType] = resourceStruct

		resourceJSON, err := json.Marshal(nsResource)

		log.Println("Resourcejson is " + string(resourceJSON))

		body, err := c.createResource(resourceType, resourceJSON)
		if err != nil {
			log.Fatal("Failed to create resource of type %s, name=%s, err=%s", resourceType, name, err)
			return "", err
		}
		_ = body
	}

	return name, nil
}

//UpdateResource updates a resource of supplied type and name
func (c *NitroClient) UpdateResource(resourceType string, name string, resourceStruct interface{}) (string, error) {

	if c.ResourceExists(resourceType, name) == true {
		nsResource := make(map[string]interface{})
		nsResource[resourceType] = resourceStruct
		resourceJSON, err := json.Marshal(nsResource)

		log.Println("Resourcejson is " + string(resourceJSON))

		body, err := c.updateResource(resourceType, name, resourceJSON)
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed to update resource of type %s, name=%s err=%s", resourceType, name, err))
			return "", err
		}
		_ = body
	}

	return name, nil
}

//DeleteResource deletes a resource of supplied type and name
func (c *NitroClient) DeleteResource(resourceType string, resourceName string) error {

	_, err := c.listResource(resourceType, resourceName)
	if err == nil { // resource exists
		log.Printf("Found resource of type %s: %s", resourceType, resourceName)
		_, err = c.deleteResource(resourceType, resourceName)
		if err != nil {
			log.Println(fmt.Sprintf("Failed to delete resourceType %s: %s, err=%s", resourceType, resourceName, err))
			return err
		}
	} else {
		log.Printf("Resource %s already deleted ", resourceName)
	}
	return nil
}

//BindResource binds the 'bindingResourceName' to the 'bindToResourceName'.
func (c *NitroClient) BindResource(bindToResourceType string, bindToResourceName string, bindingResourceType string, bindingResourceName string, bindingStruct interface{}) error {
	if c.ResourceExists(bindToResourceType, bindToResourceName) == false {
		return fmt.Errorf("BindTo Resource %s of type %s does not exist", bindToResourceType, bindToResourceName)
	}

	if c.ResourceExists(bindingResourceType, bindingResourceName) == false {
		return fmt.Errorf("Binding Resource %s of type %s does not exist", bindingResourceType, bindingResourceName)
	}
	bindingName := bindToResourceType + "_" + bindingResourceType + "_binding"
	nsBinding := make(map[string]interface{})
	nsBinding[bindingName] = bindingStruct

	resourceJSON, err := json.Marshal(nsBinding)

	body, err := c.createResource(bindingName, resourceJSON)
	if err != nil {
		log.Fatal("Failed to bind resource %s to resource %s, err=%s", bindToResourceName, bindingResourceName, err)
		return err
	}
	_ = body
	return nil
}

//UnbindResource unbinds 'boundResourceName' from 'boundToResource'
func (c *NitroClient) UnbindResource(boundToResourceType string, boundToResourceName string, boundResourceType string, boundResourceName string, bindingFilterName string) error {

	if c.ResourceExists(boundToResourceType, boundToResourceName) == false {
		log.Println(fmt.Sprintf("Unbind: BoundTo Resource %s of type %s does not exist", boundToResourceType, boundToResourceName))
		return nil
	}

	if c.ResourceExists(boundResourceType, boundResourceName) == false {
		log.Println("Unbind: Bound Resource %s of type %s does not exist", boundResourceType, boundResourceName)
		return nil
	}

	_, err := c.unbindResource(boundToResourceType, boundToResourceName, boundResourceType, boundResourceName, bindingFilterName)
	if err != nil {
		return fmt.Errorf("Failed to unbind  %s:%s from %s:%s, err=%s", boundResourceType, boundResourceName, boundToResourceType, boundToResourceName, err)
	}

	return nil
}

//ResourceExists returns true if supplied resource name and type exists
func (c *NitroClient) ResourceExists(resourceType string, resourceName string) bool {
	_, err := c.listResource(resourceType, resourceName)
	if err != nil {
		log.Printf("No %s %s found", resourceType, resourceName)
		return false
	}
	log.Printf("%s %s is already present", resourceType, resourceName)
	return true
}

//FindResource returns the config of the supplied resource name and type if it exists
func (c *NitroClient) FindResource(resourceType string, resourceName string) (map[string]interface{}, error) {
	var data map[string]interface{}
	result, err := c.listResource(resourceType, resourceName)
	if err != nil {
		log.Printf("No %s %s found", resourceType, resourceName)
		return data, fmt.Errorf("No resource %s of type %s found", resourceName, resourceType)
	}
	if err = json.Unmarshal(result, &data); err != nil {
		log.Println("Failed to unmarshal Netscaler Response!")
		return data, fmt.Errorf("Failed to unmarshal Netscaler Response:resource %s of type %s", resourceName, resourceType)
	}
	if data[resourceType] == nil {
		log.Printf("No %s %s found", resourceType, resourceName)
		return data, fmt.Errorf("No resource %s of type %s found", resourceName, resourceType)
	}
	resource := data[resourceType].([]interface{})[0] //only one resource obviously

	return resource.(map[string]interface{}), nil
}

//FindAllResources finds all config objects of the supplied resource type and returns them in an array
func (c *NitroClient) FindAllResources(resourceType string) ([]map[string]interface{}, error) {
	var data map[string]interface{}
	result, err := c.listResource(resourceType, "")
	if err != nil {
		log.Printf("No %s objects found", resourceType)
		return make([]map[string]interface{}, 0, 0), nil
	}
	if err = json.Unmarshal(result, &data); err != nil {
		log.Println("Failed to unmarshal Netscaler Response!")
		return nil, fmt.Errorf("Failed to unmarshal Netscaler Response: of type %s", resourceType)
	}
	if data[resourceType] == nil {
		log.Printf("No %s found", resourceType)
		return make([]map[string]interface{}, 0, 0), nil
	}
	resources := data[resourceType].([]interface{})

	ret := make([]map[string]interface{}, len(resources), len(resources))
	for i, v := range resources {
		ret[i] = v.(map[string]interface{})
	}

	return ret, nil
}

//ResourceBindingExists returns true if the supplied binding exists
func (c *NitroClient) ResourceBindingExists(resourceType string, resourceName string, boundResourceType string, boundResourceFilterName string, boundResourceFilterValue string) bool {
	result, err := c.listBoundResources(resourceName, resourceType, boundResourceType, boundResourceFilterName, boundResourceFilterValue)
	if err != nil {
		log.Printf("No %s %s to %s %s binding found", resourceType, resourceName, boundResourceType, boundResourceFilterValue)
		return false
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		log.Println("Failed to unmarshal Netscaler Response!")
		return false
	}
	if data[fmt.Sprintf("%s_%s_binding", resourceType, boundResourceType)] == nil {
		return false
	}

	log.Printf("%s of type  %s is bound to %s type and name %s", resourceType, resourceName, boundResourceType, boundResourceFilterValue)
	return true
}

//FindBoundResource finds a bound resource if it exists
func (c *NitroClient) FindBoundResource(resourceType string, resourceName string, boundResourceType string, boundResourceFilterName string, boundResourceFilterValue string) (map[string]interface{}, error) {
	result, err := c.listBoundResources(resourceName, resourceType, boundResourceType, boundResourceFilterName, boundResourceFilterValue)
	if err != nil {
		log.Printf("No %s %s to %s %s binding found", resourceType, resourceName, boundResourceType, boundResourceFilterValue)
		return nil, fmt.Errorf("No %s %s to %s %s binding found, err=%s", resourceType, resourceName, boundResourceType, boundResourceFilterValue, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		log.Println("Failed to unmarshal Netscaler Response!")
		return nil, fmt.Errorf("Failed to unmarshal Netscaler Response!, err=%s", err)
	}
	bindingType := fmt.Sprintf("%s_%s_binding", resourceType, boundResourceType)
	if data[bindingType] == nil {
		return nil, fmt.Errorf("No %s %s to %s %s binding found", resourceType, resourceName, boundResourceType, boundResourceFilterValue)
	}
	resource := data[bindingType].([]interface{})[0] //only one resource obviously
	return resource.(map[string]interface{}), nil

}

//EnableFeatures enables the provided list of features. Depending on the licensing of the NetScaler, not all supplied features may actually
//enabled
func (c *NitroClient) EnableFeatures(featureNames []string) error {
	/* construct this:
	{
	        "nsfeature":
		{
		    "feature": [ "LB", ]
		}
	}
	*/
	featureStruct := make(map[string]map[string][]string)
	featureStruct["nsfeature"] = make(map[string][]string)
	featureStruct["nsfeature"]["feature"] = featureNames

	featureJSON, err := json.Marshal(featureStruct)
	if err != nil {
		log.Println("Failed to marshal features to JSON")
		return fmt.Errorf("Failed to marshal features to JSON")
	}

	_, err = c.enableFeatures(featureJSON)
	if err != nil {
		return fmt.Errorf("Failed to enable feature ", err)
	}
	return nil
}

//ListEnabledFeatures returns a string array of the list of features enabled on the NetScaler appliance
func (c *NitroClient) ListEnabledFeatures() ([]string, error) {

	bytes, err := c.listEnabledFeatures()
	if err != nil {
		return []string{}, fmt.Errorf("Failed to enable feature ", err)
	}
	var data map[string]interface{}
	if err = json.Unmarshal(bytes, &data); err != nil {
		log.Println("Failed to unmarshal Netscaler Response!")
		return []string{}, fmt.Errorf("Failed to unmarshal Netscaler Response to list Features")
	}
	if data["nsfeature"] == nil {
		log.Printf("No features found")
		return []string{}, fmt.Errorf("No features found")
	}
	features := data["nsfeature"].(map[string]interface{})
	// since the returned JSON map mixes boolean and array values, the unmarshal fails to figure out there
	// is an array. So we have to convert it to a string and then parse it.
	// this doesn't work: return features["feature"].([]string), nil
	// convert to string: [LB CS SSL] (note: no commas)

	result := fmt.Sprintf("%v", features["feature"])
	result = strings.TrimPrefix(result, "[")
	result = strings.TrimSuffix(result, "]")
	flist := strings.Split(result, " ")
	log.Println("result: ", result, "flist: ", flist)
	return flist, nil
}

//SaveConfig persists the config on the NetScaler to the NetScaler's persistent storage. This could take a few seconds
func (c *NitroClient) SaveConfig() error {
	/* construct this:
	{
	        "nsconfig": {}
	}
	*/
	saveStruct := make(map[string]interface{})
	saveStruct["nsconfig"] = make(map[string]interface{})

	saveJSON, err := json.Marshal(saveStruct)
	if err != nil {
		log.Println("Failed to marshal save config to JSON")
		return fmt.Errorf("Failed to marshal save config to JSON")
	}
	log.Println("saveJSON is " + string(saveJSON))

	err = c.saveConfig(saveJSON)
	if err != nil {
		return fmt.Errorf("Failed to save config ", err)
	}
	return nil
}
