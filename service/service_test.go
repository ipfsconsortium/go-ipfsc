package service

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	shell "github.com/adriamb/go-ipfs-api"
	"github.com/ipfsconsortium/go-ipfsc/storage"
	"github.com/stretchr/testify/assert"
)

type IPFSEntry struct {
	data  string
	links []string
}

type IPFSMock struct {
	dag map[string]*shell.IpfsObject
	pin map[string]bool
}

func NewIPFSMock() *IPFSMock {
	return &IPFSMock{
		make(map[string]*shell.IpfsObject),
		make(map[string]bool),
	}
}

func (m *IPFSMock) Cat(path string) (io.ReadCloser, error) {
	entry, ok := m.dag[path]
	if !ok || entry == nil {
		return nil, errors.New("Invalid path")
	}
	if entry.Data == "" {
		return nil, errors.New("No data to cat")
	}
	return ioutil.NopCloser(strings.NewReader(entry.Data)), nil
}

func (m *IPFSMock) Add(r io.Reader) (string, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return m.addFileEntry(string(content)), nil
}

func (m *IPFSMock) addFileEntry(data string) string {
	ipfshash := "/ipfs/" + data
	m.dag[ipfshash] = &shell.IpfsObject{
		Data:  data,
		Links: []shell.ObjectLink{},
	}
	return ipfshash
}

func (m *IPFSMock) addFailingEntry(name string) string {
	ipfshash := "/ipfs/" + name
	m.dag[ipfshash] = nil
	return ipfshash
}

func (m *IPFSMock) addFolderEntry(link1, link2 string) string {
	ipfshash := "/ipfs/[" + link1 + "+" + link2 + "]"
	m.dag[ipfshash] = &shell.IpfsObject{
		Data: "",
		Links: []shell.ObjectLink{
			shell.ObjectLink{Name: "1", Hash: link1, Size: 1},
			shell.ObjectLink{Name: "2", Hash: link2, Size: 1},
		},
	}
	return ipfshash
}

func (m *IPFSMock) ObjectGet(path string) (*shell.IpfsObject, error) {
	entry, ok := m.dag[path]
	if !ok || entry == nil {
		return nil, errors.New("Invalid path")
	}
	return entry, nil
}

func (m *IPFSMock) Pin(path string, recursive bool) error {
	entry, ok := m.dag[path]
	if !ok || entry == nil {
		return errors.New("Invalid path")
	}
	m.pin[path] = true
	return nil
}

func (m *IPFSMock) isPinned(path string) bool {
	yes, ok := m.pin[path]
	return yes && ok
}

func (m *IPFSMock) Unpin(path string) error {
	_, ok := m.dag[path]
	if !ok {
		return errors.New("Invalid path")
	}
	m.pin[path] = false
	return nil
}

type ENSMock struct {
	entries map[string]string
}

func NewENSMock() *ENSMock {
	return &ENSMock{
		entries: make(map[string]string),
	}
}

func (m *ENSMock) Info(name string) (string, error) {
	_, ok := m.entries[name]
	return fmt.Sprint("ENS ", name, " exists = ", ok), nil
}

func (m *ENSMock) Text(name, key string) (string, error) {
	text, ok := m.entries[name+":"+key]
	if !ok {
		return "", errors.New("Undefined ENS key")
	}
	return text, nil

}

func (m *ENSMock) SetText(name, key, text string) error {
	m.entries[name+":"+key] = text
	return nil
}

func createMockService(t *testing.T) (service *Service, ipfs *IPFSMock, ens *ENSMock) {
	ipfs = NewIPFSMock()
	ens = NewENSMock()
	ipfsc := NewIPFSCClient(ipfs, ens)

	tmp, err := ioutil.TempDir("", "dbtest")
	assert.Nil(t, err)
	s, err := storage.New(tmp)
	assert.Nil(t, err)
	err = s.SetGlobals(storage.GlobalsEntry{
		CurrentQuota: 0,
	})
	assert.Nil(t, err)

	return NewService(ipfsc, s), ipfs, ens
}

func TestPinningSync(t *testing.T) {
	s, ipfs, _ := createMockService(t)
	h1 := ipfs.addFileEntry("h1")
	h2 := ipfs.addFileEntry("h2")

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1, h2}})

	pinned, unpinned, errors, err := s.Sync([]string{"set1.eth"})
	assert.Equal(t, 2, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	assert.True(t, ipfs.isPinned(h1))
	assert.True(t, ipfs.isPinned(h2))
}

func TestConsortiumSync(t *testing.T) {
	s, ipfs, _ := createMockService(t)
	h1 := ipfs.addFileEntry("h1")
	h2 := ipfs.addFileEntry("h2")
	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1, h2}})

	h3 := ipfs.addFileEntry("h3")
	s.ipfsc.WritePinningManifest("set2.eth", &PinningManifest{Pin: []string{h2, h3}})

	s.ipfsc.WriteConsortiumManifest("consortium.eth", &ConsortiumManifest{
		Members: []ConsortiumMember{
			ConsortiumMember{EnsName: "set1.eth"},
			ConsortiumMember{EnsName: "set2.eth"},
		},
	})

	pinned, unpinned, errors, err := s.Sync([]string{"consortium.eth"})
	assert.Equal(t, 3, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	assert.True(t, ipfs.isPinned(h1))
	assert.True(t, ipfs.isPinned(h2))
	assert.True(t, ipfs.isPinned(h3))
}

func TestDirSync(t *testing.T) {
	s, ipfs, _ := createMockService(t)
	h11 := ipfs.addFileEntry("h11")
	h121 := ipfs.addFileEntry("h121")
	h122 := ipfs.addFileEntry("h122")
	h12 := ipfs.addFolderEntry(h121, h122)
	h1 := ipfs.addFolderEntry(h11, h12)

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1}})
	pinned, unpinned, errors, err := s.Sync([]string{"set1.eth"})
	assert.Equal(t, 5, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	assert.True(t, ipfs.isPinned(h1))
	assert.True(t, ipfs.isPinned(h11))
	assert.True(t, ipfs.isPinned(h12))
	assert.True(t, ipfs.isPinned(h121))
	assert.True(t, ipfs.isPinned(h122))
}

func TestUpdateSimple(t *testing.T) {
	s, ipfs, _ := createMockService(t)
	h1 := ipfs.addFileEntry("h1")
	h2 := ipfs.addFileEntry("h2")
	h3 := ipfs.addFileEntry("h3")

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1, h2}})
	pinned, unpinned, errors, err := s.Sync([]string{"set1.eth"})
	assert.Equal(t, 2, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h2, h3}})
	pinned, unpinned, errors, err = s.Sync([]string{"set1.eth"})
	assert.Equal(t, 1, pinned)
	assert.Equal(t, 1, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	assert.False(t, ipfs.isPinned(h1))
	assert.True(t, ipfs.isPinned(h2))
	assert.True(t, ipfs.isPinned(h3))
}

func TestUpdateSharedBranches(t *testing.T) {
	s, ipfs, _ := createMockService(t)

	/*

		             h1     h2
					/  \   /  \
				  h11   h12    h21
				        /  \
		              h121  h122
	*/

	h11 := ipfs.addFileEntry("h11")
	h121 := ipfs.addFileEntry("h121")
	h122 := ipfs.addFileEntry("h122")
	h12 := ipfs.addFolderEntry(h121, h122)
	h1 := ipfs.addFolderEntry(h11, h12)
	h21 := ipfs.addFileEntry("h21")
	h2 := ipfs.addFolderEntry(h21, h11)

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1}})
	s.ipfsc.WritePinningManifest("set2.eth", &PinningManifest{Pin: []string{h2}})
	pinned, unpinned, errors, err := s.Sync([]string{"set1.eth", "set2.eth"})
	assert.Equal(t, 7, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	pinned, unpinned, errors, err = s.Sync([]string{"set1.eth"})
	assert.Equal(t, 0, pinned)
	assert.Equal(t, 2, unpinned)
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	assert.True(t, ipfs.isPinned(h1))
	assert.True(t, ipfs.isPinned(h11))
	assert.True(t, ipfs.isPinned(h12))
	assert.True(t, ipfs.isPinned(h121))
	assert.True(t, ipfs.isPinned(h122))
	assert.False(t, ipfs.isPinned(h21))
	assert.False(t, ipfs.isPinned(h2))
}

func TestNounpinWhenFail(t *testing.T) {
	s, ipfs, _ := createMockService(t)
	h1 := ipfs.addFileEntry("h1")
	h2 := ipfs.addFileEntry("h2")
	h3 := ipfs.addFileEntry("h3")
	hfail := ipfs.addFailingEntry("fail1")

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1, h2}})
	_, _, errors, err := s.Sync([]string{"set1.eth"})
	assert.Equal(t, 0, errors)
	assert.Nil(t, err)

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1, hfail, h3}})
	pinned, unpinned, errors, err := s.Sync([]string{"set1.eth"})
	assert.Equal(t, 1, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Equal(t, 1, errors)
	assert.Nil(t, err)
}

func TestReappearHash(t *testing.T) {
	s, ipfs, _ := createMockService(t)
	h1 := ipfs.addFileEntry("h1")

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1}})
	pinned, unpinned, errors, err := s.Sync([]string{"set1.eth"})
	assert.Equal(t, 0, errors)
	assert.Equal(t, 1, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Nil(t, err)

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{}})
	pinned, unpinned, errors, err = s.Sync([]string{"set1.eth"})
	assert.Equal(t, 0, errors)
	assert.Equal(t, 0, pinned)
	assert.Equal(t, 1, unpinned)
	assert.Nil(t, err)

	s.ipfsc.WritePinningManifest("set1.eth", &PinningManifest{Pin: []string{h1}})
	pinned, unpinned, errors, err = s.Sync([]string{"set1.eth"})
	assert.Equal(t, 0, errors)
	assert.Equal(t, 1, pinned)
	assert.Equal(t, 0, unpinned)
	assert.Nil(t, err)
}
