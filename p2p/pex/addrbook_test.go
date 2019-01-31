package pex

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
)

func createTempFileName(prefix string) string {
	f, err := ioutil.TempFile("", prefix)
	if err != nil {
		panic(err)
	}
	fname := f.Name()
	err = f.Close()
	if err != nil {
		panic(err)
	}
	return fname
}

func deleteTempFile(fname string) {
	err := os.Remove(fname)
	if err != nil {
		panic(err)
	}
}

func TestAddrBookGetSelectionWithOneMarkedGood(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	assert.Zero(t, book.Size())

	N := 10
	randAddrs := randNetAddressPairs(t, N)
	for _, addr := range randAddrs {
		book.AddAddress(addr.addr, addr.src)
	}

	// mark one addr as good
	for _, addr := range randAddrs[:1] {
		book.MarkGood(addr.addr)
	}

	// ensure we can get selection.
	// this used to result in an infinite loop (!)
	addrs := book.GetSelectionWithBias(biasToSelectNewPeers)
	assert.NotNil(t, addrs, "expected an address")
}

func TestAddrBookGetSelectionWithOneNotMarkedGood(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	// 0 addresses
	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	assert.Zero(t, book.Size())

	N := 10
	randAddrs := randNetAddressPairs(t, N)
	// mark all addrs but one as good
	for _, addr := range randAddrs[:N-1] {
		book.AddAddress(addr.addr, addr.src)
		book.MarkGood(addr.addr)
	}
	// add the last addr, don't mark as good
	for _, addr := range randAddrs[N-1:] {
		book.AddAddress(addr.addr, addr.src)
	}

	// ensure we can get selection.
	// this used to result in an infinite loop (!)
	addrs := book.GetSelectionWithBias(biasToSelectNewPeers)
	assert.NotNil(t, addrs, "expected an address")
}

func TestAddrBookAddressSelection(t *testing.T) {
	booksizes := []int{10, 100, 320}
	// for all book sizes above
	for _, b := range(booksizes) {
		// generate all possible combinations of old addresses (i) and new addresses (b-i)
		for i := 0; i < b+1; i++ {
			book := createAddrBookAddressSelection(t, i, b-i)
			addrs := book.GetSelectionWithBias(biasToSelectNewPeers)
			assert.NotNil(t, addrs, "expected an address")

			// TODO - verify that the number of old and new addresses are the ones expected
			// TODO - verify that the order of the addresses is correct
			oldaddrs, newaddrs, layout := countOldAndNewInAddrs(addrs, book)
			_ = oldaddrs
			_ = newaddrs
			_ = layout
			//fmt.Printf("\nold: %d, new: %d, Num old returned %d, Num new returned %d\n", i, b-i, oldaddr, newaddr)
			//fmt.Println("Here is the layout", layout)
		}
	}
}

// Counts the number of Old and New addresses returned and returns the order in the layout
func countOldAndNewInAddrs(addrs []*p2p.NetAddress, book *addrBook) (nold, nnew int, layout []string) {
	nold = 0
	nnew = 0
	layout = make([]string, len(addrs))

	for idx, addr := range(addrs) {
		layout[idx] = fmt.Sprintf("X%d", idx+1)
		if addr == nil {
			continue
		}
		if book.IsGood(addr) {
			nold++
			layout[idx] = fmt.Sprintf("O%d", idx+1)

		} else {
			nnew++
			layout[idx] = fmt.Sprintf("N%d", idx+1)

		}
	}
	return nold, nnew, layout
}

// Creates a Book with m old and n new addresses.
func createAddrBookAddressSelection(t *testing.T, m, n int) *addrBook {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	assert.Zero(t, book.Size())

	// create m entries and mark all old
	randAddrs := randNetAddressPairs(t, m)
	for _, addr := range randAddrs {
		book.AddAddress(addr.addr, addr.src)
		book.MarkGood(addr.addr)
	}
	// create n entries and leave all new
	randAddrs = randNetAddressPairs(t, n)
	for _, addr := range randAddrs {
		book.AddAddress(addr.addr, addr.src)
	}

	return book
}


func TestAddrBookPickAddress(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	// 0 addresses
	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	assert.Zero(t, book.Size())

	addr := book.PickAddress(50)
	assert.Nil(t, addr, "expected no address")

	randAddrs := randNetAddressPairs(t, 1)
	addrSrc := randAddrs[0]
	book.AddAddress(addrSrc.addr, addrSrc.src)

	// pick an address when we only have new address
	addr = book.PickAddress(0)
	assert.NotNil(t, addr, "expected an address")
	addr = book.PickAddress(50)
	assert.NotNil(t, addr, "expected an address")
	addr = book.PickAddress(100)
	assert.NotNil(t, addr, "expected an address")

	// pick an address when we only have old address
	book.MarkGood(addrSrc.addr)
	addr = book.PickAddress(0)
	assert.NotNil(t, addr, "expected an address")
	addr = book.PickAddress(50)
	assert.NotNil(t, addr, "expected an address")

	// in this case, nNew==0 but we biased 100% to new, so we return nil
	addr = book.PickAddress(100)
	assert.Nil(t, addr, "did not expected an address")
}

func TestAddrBookSaveLoad(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	// 0 addresses
	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	book.saveToFile(fname)

	book = NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	book.loadFromFile(fname)

	assert.Zero(t, book.Size())

	// 100 addresses
	randAddrs := randNetAddressPairs(t, 100)

	for _, addrSrc := range randAddrs {
		book.AddAddress(addrSrc.addr, addrSrc.src)
	}

	assert.Equal(t, 100, book.Size())
	book.saveToFile(fname)

	book = NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	book.loadFromFile(fname)

	assert.Equal(t, 100, book.Size())
}

func TestAddrBookLookup(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	randAddrs := randNetAddressPairs(t, 100)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	for _, addrSrc := range randAddrs {
		addr := addrSrc.addr
		src := addrSrc.src
		book.AddAddress(addr, src)

		ka := book.addrLookup[addr.ID]
		assert.NotNil(t, ka, "Expected to find KnownAddress %v but wasn't there.", addr)

		if !(ka.Addr.Equals(addr) && ka.Src.Equals(src)) {
			t.Fatalf("KnownAddress doesn't match addr & src")
		}
	}
}

func TestAddrBookPromoteToOld(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	randAddrs := randNetAddressPairs(t, 100)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	for _, addrSrc := range randAddrs {
		book.AddAddress(addrSrc.addr, addrSrc.src)
	}

	// Attempt all addresses.
	for _, addrSrc := range randAddrs {
		book.MarkAttempt(addrSrc.addr)
	}

	// Promote half of them
	for i, addrSrc := range randAddrs {
		if i%2 == 0 {
			book.MarkGood(addrSrc.addr)
		}
	}

	// TODO: do more testing :)

	selection := book.GetSelection()
	t.Logf("selection: %v", selection)

	if len(selection) > book.Size() {
		t.Errorf("selection could not be bigger than the book")
	}

	selection = book.GetSelectionWithBias(30)
	t.Logf("selection: %v", selection)

	if len(selection) > book.Size() {
		t.Errorf("selection with bias could not be bigger than the book")
	}

	assert.Equal(t, book.Size(), 100, "expecting book size to be 100")
}

func TestAddrBookHandlesDuplicates(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())

	randAddrs := randNetAddressPairs(t, 100)

	differentSrc := randIPv4Address(t)
	for _, addrSrc := range randAddrs {
		book.AddAddress(addrSrc.addr, addrSrc.src)
		book.AddAddress(addrSrc.addr, addrSrc.src)  // duplicate
		book.AddAddress(addrSrc.addr, differentSrc) // different src
	}

	assert.Equal(t, 100, book.Size())
}

type netAddressPair struct {
	addr *p2p.NetAddress
	src  *p2p.NetAddress
}

func randNetAddressPairs(t *testing.T, n int) []netAddressPair {
	randAddrs := make([]netAddressPair, n)
	for i := 0; i < n; i++ {
		randAddrs[i] = netAddressPair{addr: randIPv4Address(t), src: randIPv4Address(t)}
	}
	return randAddrs
}

func randIPv4Address(t *testing.T) *p2p.NetAddress {
	for {
		ip := fmt.Sprintf("%v.%v.%v.%v",
			cmn.RandIntn(254)+1,
			cmn.RandIntn(255),
			cmn.RandIntn(255),
			cmn.RandIntn(255),
		)
		port := cmn.RandIntn(65535-1) + 1
		id := p2p.ID(hex.EncodeToString(cmn.RandBytes(p2p.IDByteLength)))
		idAddr := p2p.IDAddressString(id, fmt.Sprintf("%v:%v", ip, port))
		addr, err := p2p.NewNetAddressString(idAddr)
		assert.Nil(t, err, "error generating rand network address")
		if addr.Routable() {
			return addr
		}
	}
}

func TestAddrBookRemoveAddress(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())

	addr := randIPv4Address(t)
	book.AddAddress(addr, addr)
	assert.Equal(t, 1, book.Size())

	book.RemoveAddress(addr)
	assert.Equal(t, 0, book.Size())

	nonExistingAddr := randIPv4Address(t)
	book.RemoveAddress(nonExistingAddr)
	assert.Equal(t, 0, book.Size())
}

func TestAddrBookGetSelection(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())

	// 1) empty book
	assert.Empty(t, book.GetSelection())

	// 2) add one address
	addr := randIPv4Address(t)
	book.AddAddress(addr, addr)

	assert.Equal(t, 1, len(book.GetSelection()))
	assert.Equal(t, addr, book.GetSelection()[0])

	// 3) add a bunch of addresses
	randAddrs := randNetAddressPairs(t, 100)
	for _, addrSrc := range randAddrs {
		book.AddAddress(addrSrc.addr, addrSrc.src)
	}

	// check there is no duplicates
	addrs := make(map[string]*p2p.NetAddress)
	selection := book.GetSelection()
	for _, addr := range selection {
		if dup, ok := addrs[addr.String()]; ok {
			t.Fatalf("selection %v contains duplicates %v", selection, dup)
		}
		addrs[addr.String()] = addr
	}

	if len(selection) > book.Size() {
		t.Errorf("selection %v could not be bigger than the book", selection)
	}
}

func TestAddrBookGetSelectionWithBias(t *testing.T) {
	const biasTowardsNewAddrs = 30

	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())

	// 1) empty book
	selection := book.GetSelectionWithBias(biasTowardsNewAddrs)
	assert.Empty(t, selection)

	// 2) add one address
	addr := randIPv4Address(t)
	book.AddAddress(addr, addr)

	selection = book.GetSelectionWithBias(biasTowardsNewAddrs)
	assert.Equal(t, 1, len(selection))
	assert.Equal(t, addr, selection[0])

	// 3) add a bunch of addresses
	randAddrs := randNetAddressPairs(t, 100)
	for _, addrSrc := range randAddrs {
		book.AddAddress(addrSrc.addr, addrSrc.src)
	}

	// check there is no duplicates
	addrs := make(map[string]*p2p.NetAddress)
	selection = book.GetSelectionWithBias(biasTowardsNewAddrs)
	for _, addr := range selection {
		if dup, ok := addrs[addr.String()]; ok {
			t.Fatalf("selection %v contains duplicates %v", selection, dup)
		}
		addrs[addr.String()] = addr
	}

	if len(selection) > book.Size() {
		t.Fatalf("selection %v could not be bigger than the book", selection)
	}

	// 4) mark 80% of the addresses as good
	randAddrsLen := len(randAddrs)
	for i, addrSrc := range randAddrs {
		if int((float64(i)/float64(randAddrsLen))*100) >= 20 {
			book.MarkGood(addrSrc.addr)
		}
	}

	selection = book.GetSelectionWithBias(biasTowardsNewAddrs)

	// check that ~70% of addresses returned are good
	good := 0
	for _, addr := range selection {
		if book.IsGood(addr) {
			good++
		}
	}
	got, expected := int((float64(good)/float64(len(selection)))*100), (100 - biasTowardsNewAddrs)
	if got >= expected {
		t.Fatalf("expected more good peers (%% got: %d, %% expected: %d, number of good addrs: %d, total: %d)", got, expected, good, len(selection))
	}
}

func TestAddrBookHasAddress(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	addr := randIPv4Address(t)
	book.AddAddress(addr, addr)

	assert.True(t, book.HasAddress(addr))

	book.RemoveAddress(addr)

	assert.False(t, book.HasAddress(addr))
}

func testCreatePrivateAddrs(t *testing.T, numAddrs int) ([]*p2p.NetAddress, []string) {
	addrs := make([]*p2p.NetAddress, numAddrs)
	for i := 0; i < numAddrs; i++ {
		addrs[i] = randIPv4Address(t)
	}

	private := make([]string, numAddrs)
	for i, addr := range addrs {
		private[i] = string(addr.ID)
	}
	return addrs, private
}

func TestAddrBookEmpty(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())
	// Check that empty book is empty
	require.True(t, book.Empty())
	// Check that book with our address is empty
	book.AddOurAddress(randIPv4Address(t))
	require.True(t, book.Empty())
	// Check that book with private addrs is empty
	_, privateIds := testCreatePrivateAddrs(t, 5)
	book.AddPrivateIDs(privateIds)
	require.True(t, book.Empty())

	// Check that book with address is not empty
	book.AddAddress(randIPv4Address(t), randIPv4Address(t))
	require.False(t, book.Empty())
}

func TestPrivatePeers(t *testing.T) {
	fname := createTempFileName("addrbook_test")
	defer deleteTempFile(fname)

	book := NewAddrBook(fname, true)
	book.SetLogger(log.TestingLogger())

	addrs, private := testCreatePrivateAddrs(t, 10)
	book.AddPrivateIDs(private)

	// private addrs must not be added
	for _, addr := range addrs {
		err := book.AddAddress(addr, addr)
		if assert.Error(t, err) {
			_, ok := err.(ErrAddrBookPrivate)
			assert.True(t, ok)
		}
	}

	// addrs coming from private peers must not be added
	err := book.AddAddress(randIPv4Address(t), addrs[0])
	if assert.Error(t, err) {
		_, ok := err.(ErrAddrBookPrivateSrc)
		assert.True(t, ok)
	}
}
