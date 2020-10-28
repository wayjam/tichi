package externalplugins

import (
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Configuration is the top-level serialization target for external plugin Configuration.
type Configuration struct {
	TiCommunityLgtm   []TiCommunityLgtm   `json:"ti-community-lgtm,omitempty"`
	TiCommunityMerge  []TiCommunityMerge  `json:"ti-community-merge,omitempty"`
	TiCommunityOwners []TiCommunityOwners `json:"ti-community-owners,omitempty"`
}

// TiCommunityLgtm specifies a configuration for a single ti community lgtm.
// The configuration for the ti community lgtm plugin is defined as a list of these structures.
type TiCommunityLgtm struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// ReviewActsAsLgtm indicates that a GitHub review of "merge" or "request changes"
	// acts as adding or removing the lgtm label.
	ReviewActsAsLgtm bool `json:"review_acts_as_lgtm,omitempty"`
	// StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
	// squashed commits before removing lgtm labels.
	StoreTreeHash bool `json:"store_tree_hash,omitempty"`
	// WARNING: This disables the security mechanism that prevents a malicious member (or
	// compromised GitHub account) from merging arbitrary code. Use with caution.
	//
	// StickyLgtmTeam specifies the GitHub team whose members are trusted with sticky LGTM,
	// which eliminates the need to re-lgtm minor fixes/updates.
	StickyLgtmTeam string `json:"trusted_team_for_sticky_lgtm,omitempty"`
	// PullOwnersURL specifies the URL of the reviewer of pull request.
	PullOwnersURL string `json:"pull_owners_url,omitempty"`
}

// TiCommunityMerge specifies a configuration for a single merge.
//
// The configuration for the merge plugin is defined as a list of these structures.
type TiCommunityMerge struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
	// squashed commits before removing can merge labels.
	StoreTreeHash bool `json:"store_tree_hash,omitempty"`
	// PullOwnersURL specifies the URL of the reviewer of pull request.
	PullOwnersURL string `json:"pull_owners_url,omitempty"`
}

// TiCommunityMerge specifies a configuration for a single owners.
//
// The configuration for the owners plugin is defined as a list of these structures.
type TiCommunityOwners struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// SigEndpoint specifies the URL of the sig info.
	SigEndpoint string `json:"sig_endpoint,omitempty"`
}

// LgtmFor finds the Lgtm for a repo, if one exists
// a trigger can be listed for the repo itself or for the
// owning organization
func (c *Configuration) LgtmFor(org, repo string) *TiCommunityLgtm {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, lgtm := range c.TiCommunityLgtm {
		if !sets.NewString(lgtm.Repos...).Has(fullName) {
			continue
		}
		return &lgtm
	}
	// If you don't find anything, loop again looking for an org config
	for _, lgtm := range c.TiCommunityLgtm {
		if !sets.NewString(lgtm.Repos...).Has(org) {
			continue
		}
		return &lgtm
	}
	return &TiCommunityLgtm{}
}

// MergeFor finds the TiCommunityMerge for a repo, if one exists.
// TiCommunityMerge configuration can be listed for a repository
// or an organization.
func (c *Configuration) MergeFor(org, repo string) *TiCommunityMerge {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, merge := range c.TiCommunityMerge {
		if !sets.NewString(merge.Repos...).Has(fullName) {
			continue
		}
		return &merge
	}
	// If you don't find anything, loop again looking for an org config
	for _, merge := range c.TiCommunityMerge {
		if !sets.NewString(merge.Repos...).Has(org) {
			continue
		}
		return &merge
	}
	return &TiCommunityMerge{}
}

// OwnersFor finds the TiCommunityOwners for a repo, if one exists.
// TiCommunityOwners configuration can be listed for a repository
// or an organization.
func (c *Configuration) OwnersFor(org, repo string) *TiCommunityOwners {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, owners := range c.TiCommunityOwners {
		if !sets.NewString(owners.Repos...).Has(fullName) {
			continue
		}
		return &owners
	}
	// If you don't find anything, loop again looking for an org config
	for _, owners := range c.TiCommunityOwners {
		if !sets.NewString(owners.Repos...).Has(org) {
			continue
		}
		return &owners
	}
	return &TiCommunityOwners{}
}

// Validate will return an error if there are any invalid external plugin config.
func (c *Configuration) Validate() error {
	if err := validateLgtm(c.TiCommunityLgtm); err != nil {
		return err
	}

	if err := validateMerge(c.TiCommunityMerge); err != nil {
		return err
	}

	if err := validateOwners(c.TiCommunityOwners); err != nil {
		return err
	}

	return nil
}

// validateLgtm will return an error if the URL configured by lgtm is invalid.
func validateLgtm(lgtms []TiCommunityLgtm) error {
	for _, lgtm := range lgtms {
		_, err := url.ParseRequestURI(lgtm.PullOwnersURL)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateMerge will return an error if the URL configured by merge is invalid.
func validateMerge(merges []TiCommunityMerge) error {
	for _, merge := range merges {
		_, err := url.ParseRequestURI(merge.PullOwnersURL)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateOwners will return an error if the endpoint configured by merge is invalid.
func validateOwners(merges []TiCommunityOwners) error {
	for _, merge := range merges {
		_, err := url.ParseRequestURI(merge.SigEndpoint)
		if err != nil {
			return err
		}
	}

	return nil
}
