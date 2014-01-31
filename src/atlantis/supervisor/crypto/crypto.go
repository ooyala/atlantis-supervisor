/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package crypto

import (
	"atlantis/crypto"
	"atlantis/supervisor/rpc/types"
	"encoding/json"
)

func EncryptAppDep(data *types.AppDep) error {
	// encrypt DataMap and nil out DataMap
	// convert to JSON
	jsonBytes, err := json.Marshal(data.DataMap)
	if err != nil {
		return err
	}
	// encrypt into Data
	data.EncryptedData = string(crypto.Encrypt(jsonBytes))
	// nil out DataMap
	data.DataMap = nil
	return nil
}

func DecryptAppDep(data *types.AppDep) error {
	// decrypt Data to DataMap
	var err error
	data.DataMap, err = DecryptedAppDepData(data)
	return err
}

func DecryptedAppDepData(data *types.AppDep) (map[string]interface{}, error) {
	// decrypt Data
	decryptedBytes := crypto.Decrypt([]byte(data.EncryptedData))
	dataMap := map[string]interface{}{}
	// Unmarshal JSON
	return dataMap, json.Unmarshal(decryptedBytes, &dataMap)
}
