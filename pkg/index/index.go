// Copyright 2021 The Kubeswitch authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package index

import (
	"fmt"
	"os"
	"time"

	"github.com/danielfoehrkn/kubeswitch/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	// indexStateFileName is the filename of the index state file containing the last time a Kind index has been updated
	// located at the root of the given kubeconfigDirectory
	indexStateFileName = "index.state"
	// indexFileName is the filename of the file containing a pre-computed context -> kubeconfig path mapping
	// located at the root of the given kubeconfigDirectory
	indexFileName = "index"
)

type SearchIndex struct {
	log                 *logrus.Entry
	indexFilepath       string
	indexStateFilepath  string
	kubeconfigStoreKind types.StoreKind
	content             *types.Index
}

// New creates a new SearchIndex
func New(log *logrus.Entry, storeKind types.StoreKind, stateDirectory string, storeID string) (*SearchIndex, error) {
	if _, err := os.Stat(stateDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(stateDirectory, 0755); err != nil {
			return nil, err
		}
	}

	indexStateFilepath := fmt.Sprintf("%s/switch.%s.%s", stateDirectory, storeID, indexStateFileName)
	indexFilepath := fmt.Sprintf("%s/switch.%s.%s", stateDirectory, storeID, indexFileName)

	i := SearchIndex{
		log:                 log,
		indexFilepath:       indexFilepath,
		indexStateFilepath:  indexStateFilepath,
		kubeconfigStoreKind: storeKind,
	}

	indexFromFile, err := i.loadFromFile()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	i.content = indexFromFile
	return &i, nil
}

func (i *SearchIndex) HasContent() bool {
	return i.content != nil
}

func (i *SearchIndex) HasKind(kind types.StoreKind) bool {
	return i.content != nil && i.content.Kind == kind
}

func (i *SearchIndex) GetContent() map[string]string {
	if i.content == nil {
		return nil
	}
	return i.content.ContextToPathMapping
}

// LoadIndexFromFile takes a filename and de-serializes the contents into an SearchIndex object.
func (i *SearchIndex) loadFromFile() (*types.Index, error) {
	// an index file is not required. Its ok if it does not exist.
	if _, err := os.Stat(i.indexFilepath); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(i.indexFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file from %q. File corrupt?: %v", i.indexFilepath, err)
	}

	index := &types.Index{}
	if len(bytes) == 0 {
		return index, nil
	}

	err = yaml.Unmarshal(bytes, &index)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal index file with path '%s': %v", i.indexFilepath, err)
	}
	return index, nil
}

// ShouldBeUsed checks if the index file with pre-computed mappings should be used
func (i *SearchIndex) ShouldBeUsed(config *types.Config, storeLocalRefreshIndexAfter *time.Duration) (bool, error) {
	indexState, err := i.getIndexState()
	if err != nil {
		return false, fmt.Errorf("failed to get index state: %v", err)
	}

	// do not read from existing index if there is no index state file
	// we do not know when the index last has been refreshed
	if indexState == nil || indexState.Kind != i.kubeconfigStoreKind {
		return false, nil
	}

	var refreshAfter *time.Duration
	if config != nil && config.RefreshIndexAfter != nil {
		refreshAfter = config.RefreshIndexAfter
	}
	if storeLocalRefreshIndexAfter != nil {
		refreshAfter = storeLocalRefreshIndexAfter
	}

	if refreshAfter == nil {
		return false, nil
	}

	return time.Now().UTC().Before(indexState.LastUpdateTime.UTC().Add(*refreshAfter)), nil
}

func (i *SearchIndex) WriteState(toWrite types.IndexState) error {
	// creates or truncate/clean the existing state file (only state is last execution anyways atm.)
	file, err := os.Create(i.indexStateFilepath)
	if err != nil {
		return err
	}
	defer file.Close()

	output, err := yaml.Marshal(toWrite)
	if err != nil {
		return err
	}

	_, err = file.Write(output)
	if err != nil {
		return err
	}

	return nil
}

func (i *SearchIndex) Write(toWrite types.Index) error {
	// creates or truncate/clean the existing file
	file, err := os.Create(i.indexFilepath)
	if err != nil {
		return err
	}
	defer file.Close()

	output, err := yaml.Marshal(toWrite)
	if err != nil {
		return err
	}

	_, err = file.Write(output)
	if err != nil {
		return err
	}

	return nil
}

func (i *SearchIndex) Delete() error {
	if _, err := os.Stat(i.indexStateFilepath); err != nil {
		if os.IsNotExist(err) {
			// occurs during first execution of the hook
			return nil
		}
		return err
	}

	if err := os.Remove(i.indexFilepath); err != nil {
		return err
	}

	if err := os.Remove(i.indexStateFilepath); err != nil {
		return err
	}

	return nil
}

// getIndexState loads and unmarshalls an index state file
func (i *SearchIndex) getIndexState() (*types.IndexState, error) {
	if _, err := os.Stat(i.indexStateFilepath); err != nil {
		if os.IsNotExist(err) {
			// occurs during first execution of the hook
			i.log.Warnf("SearchIndex state file not found under path: %q", i.indexStateFilepath)
			return nil, nil
		}
		return nil, err
	}

	bytes, err := os.ReadFile(i.indexStateFilepath)
	if err != nil {
		return nil, err
	}

	state := &types.IndexState{}
	if len(bytes) == 0 {
		return state, nil
	}

	err = yaml.Unmarshal(bytes, &state)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal index state file with path '%s': %v", i.indexStateFilepath, err)
	}

	return state, nil
}
