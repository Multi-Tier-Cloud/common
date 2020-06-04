/* Copyright 2020 Multi-Tier-Cloud Development Team
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

package util

import (
	"context"
	"os"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	"github.com/ipfs/go-ipfs-files"
	dag "github.com/ipfs/go-merkledag"
	dagtest "github.com/ipfs/go-merkledag/test"
	"github.com/ipfs/go-mfs"
	"github.com/ipfs/go-unixfs"
)

func IpfsHashBytes(data []byte) (hash string, err error) {
	bytesFile := files.NewBytesFile(data)
	return getIpfsHash(bytesFile)
}

func IpfsHashFile(path string) (hash string, err error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	fileNode, err := files.NewSerialFile(path, false, stat)
	if err != nil {
		return "", err
	}
	defer fileNode.Close()

	return getIpfsHash(fileNode)
}

func getIpfsHash(fileNode files.Node) (hash string, err error) {
	ctx := context.Background()
	nilIpfsNode, err := core.NewNode(ctx, &core.BuildCfg{NilRepo: true})
	if err != nil {
		return "", err
	}

	bserv := blockservice.New(nilIpfsNode.Blockstore, nilIpfsNode.Exchange)
	dserv := dag.NewDAGService(bserv)

	fileAdder, err := coreunix.NewAdder(
		ctx, nilIpfsNode.Pinning, nilIpfsNode.Blockstore, dserv)
	if err != nil {
		return "", err
	}

	fileAdder.Pin = false
	fileAdder.CidBuilder = dag.V0CidPrefix()

	mockDserv := dagtest.Mock()
	emptyDirNode := unixfs.EmptyDirNode()
	emptyDirNode.SetCidBuilder(fileAdder.CidBuilder)
	mfsRoot, err := mfs.NewRoot(ctx, mockDserv, emptyDirNode, nil)
	if err != nil {
		return "", err
	}
	fileAdder.SetMfsRoot(mfsRoot)

	dagIpldNode, err := fileAdder.AddAllAndPin(fileNode)
	if err != nil {
		return "", err
	}

	hash = dagIpldNode.String()
	return hash, nil
}