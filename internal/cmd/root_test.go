package cmd

import "testing"

func TestBridgeAliasesResolveToClaimCommands(t *testing.T) {
	root := NewRootCmd()

	cases := []struct {
		alias string
		want  string
	}{
		{alias: "bridge-eth", want: "claim-eth"},
		{alias: "bridge-erc20", want: "claim-erc20"},
		{alias: "bridge-erc721", want: "claim-erc721"},
		{alias: "bridge-erc1155", want: "claim-erc1155"},
	}

	for _, tc := range cases {
		cmd, _, err := root.Find([]string{tc.alias})
		if err != nil {
			t.Fatalf("Find(%s) error: %v", tc.alias, err)
		}
		if cmd.Name() != tc.want {
			t.Fatalf("alias %s resolved to %s, want %s", tc.alias, cmd.Name(), tc.want)
		}
	}
}
