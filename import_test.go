package iavl

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/tendermint/tm-db"
)

func TestImporter_NegativeVersion(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	_, err = tree.Import(-1)
	require.Error(t, err)
}

func TestImporter_NotEmpty(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	tree.Set([]byte("a"), []byte{1})
	_, _, err = tree.SaveVersion()
	require.NoError(t, err)

	_, err = tree.Import(1)
	require.Error(t, err)
}

func TestImporter_NotEmptyDatabase(t *testing.T) {
	db := db.NewMemDB()

	tree, err := NewMutableTree(db, 0)
	require.NoError(t, err)
	tree.Set([]byte("a"), []byte{1})
	_, _, err = tree.SaveVersion()
	require.NoError(t, err)

	tree, err = NewMutableTree(db, 0)
	require.NoError(t, err)
	_, err = tree.Load()
	require.NoError(t, err)

	_, err = tree.Import(1)
	require.Error(t, err)
}

func TestImporter_NotEmptyUnsaved(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	tree.Set([]byte("a"), []byte{1})

	_, err = tree.Import(1)
	require.Error(t, err)
}

func TestImporter_Add(t *testing.T) {
	k := []byte("key")
	v := []byte("value")

	testcases := map[string]struct {
		node  *ExportNode
		valid bool
	}{
		"nil node":          {nil, false},
		"valid":             {&ExportNode{Key: k, Value: v, Version: 1, Height: 0}, true},
		"no key":            {&ExportNode{Key: nil, Value: v, Version: 1, Height: 0}, false},
		"no value":          {&ExportNode{Key: k, Value: nil, Version: 1, Height: 0}, false},
		"version too large": {&ExportNode{Key: k, Value: v, Version: 2, Height: 0}, false},
		"no version":        {&ExportNode{Key: k, Value: v, Version: 0, Height: 0}, false},
		// further cases will be handled by Node.validate()
	}
	for desc, tc := range testcases {
		tc := tc // appease scopelint
		t.Run(desc, func(t *testing.T) {
			tree, err := NewMutableTree(db.NewMemDB(), 0)
			require.NoError(t, err)
			importer, err := tree.Import(1)
			require.NoError(t, err)
			defer importer.Close()

			err = importer.Add(tc.node)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestImporter_Add_Closed(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	importer, err := tree.Import(1)
	require.NoError(t, err)

	importer.Close()
	err = importer.Add(&ExportNode{Key: []byte("key"), Value: []byte("value"), Version: 1, Height: 0})
	require.Error(t, err)
	require.Equal(t, ErrNoImport, err)
}

func TestImporter_Close(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	importer, err := tree.Import(1)
	require.NoError(t, err)

	err = importer.Add(&ExportNode{Key: []byte("key"), Value: []byte("value"), Version: 1, Height: 0})
	require.NoError(t, err)

	importer.Close()
	has := tree.Has([]byte("key"))
	require.False(t, has)

	importer.Close()
}

func TestImporter_Commit(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	importer, err := tree.Import(1)
	require.NoError(t, err)

	err = importer.Add(&ExportNode{Key: []byte("key"), Value: []byte("value"), Version: 1, Height: 0})
	require.NoError(t, err)

	err = importer.Commit()
	require.NoError(t, err)
	has := tree.Has([]byte("key"))
	require.True(t, has)
}

func TestImporter_Commit_Closed(t *testing.T) {
	tree, err := NewMutableTree(db.NewMemDB(), 0)
	require.NoError(t, err)
	importer, err := tree.Import(1)
	require.NoError(t, err)

	err = importer.Add(&ExportNode{Key: []byte("key"), Value: []byte("value"), Version: 1, Height: 0})
	require.NoError(t, err)

	importer.Close()
	err = importer.Commit()
	require.Error(t, err)
	require.Equal(t, ErrNoImport, err)
}

func BenchmarkImport(b *testing.B) {
	tree := setupExportTreeSized(b, 4096)
	exported := make([]*ExportNode, 0, 4096)
	exporter := tree.Export()
	for {
		item, err := exporter.Next()
		if err == io.EOF {
			break
		}
		require.NoError(b, err)
		exported = append(exported, item)
	}
	exporter.Close()

	for n := 0; n < b.N; n++ {
		newTree, err := NewMutableTree(db.NewMemDB(), 0)
		require.NoError(b, err)
		importer, err := newTree.Import(tree.Version())
		require.NoError(b, err)
		for _, item := range exported {
			err = importer.Add(item)
			require.NoError(b, err)
		}
		err = importer.Commit()
		require.NoError(b, err)
	}
}
