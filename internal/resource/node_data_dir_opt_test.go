package resource

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NodeDataDirServerOpt", func() {
	It("returns an empty string when no path is set", func() {
		Expect(NodeDataDirServerOpt("")).To(BeEmpty())
	})
	It("returns the node data path server option", func() {
		Expect(NodeDataDirServerOpt("/mnt/node-data-dir")).To(Equal(" -Dteamcity.node.data.path=/mnt/node-data-dir"))
	})
})
