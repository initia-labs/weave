package cosmosutils

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/registry"
)

const (
	PolkachuBaseURL     string = "https://www.polkachu.com"
	PolkachuSnapshotURL        = PolkachuBaseURL + "/%s"
	PolkachuChainAPI           = PolkachuBaseURL + "/api/v2/chains/%s"
	PolkachuPeersAPI           = PolkachuBaseURL + "/api/v2/chains/%s/live_peers"

	PolkachuAddrBookAPI = "https://snapshots.polkachu.com/%saddrbook/%s/addrbook.json"

	DefaultInitiaPolkachuName    string = "initia"
	DefaultSnapshotFileExtension string = ".tar.lz4"
)

type PolkachuChainAPIResponse struct {
	PolkachuServices struct {
		StateSync struct {
			Active bool   `json:"active"`
			Node   string `json:"node"`
		} `json:"state_sync"`
	} `json:"polkachu_services"`
}

type PolkachuPeersAPIResponse struct {
	PolkachuPeer string   `json:"polkachu_peer"`
	LivePeers    []string `json:"live_peers"`
}

func getPolkachuQueryParams(chainType registry.ChainType) map[string]string {
	queryParams := make(map[string]string)
	if chainType == registry.InitiaL1Testnet {
		queryParams["type"] = "testnet"
	}
	return queryParams
}

func FetchPolkachuSnapshotDownloadURL(chainSlug string) (string, error) {
	httpClient := client.NewHTTPClient()
	body, err := httpClient.Get(fmt.Sprintf(PolkachuSnapshotURL, chainSlug), "", nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var downloadURL string
	doc.Find("a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		href, exists := s.Attr("href")
		if exists && isSnapshotURL(href) {
			downloadURL = href
			return false
		}
		return true
	})

	if downloadURL == "" {
		return "", fmt.Errorf("no download URL found")
	}

	return downloadURL, nil
}

func isSnapshotURL(href string) bool {
	fileLen := len(href) - len(DefaultSnapshotFileExtension)
	if fileLen < 0 {
		return false
	}

	return href != "" && href[fileLen:] == DefaultSnapshotFileExtension
}

func FetchPolkachuStateSyncURL(chainType registry.ChainType) (string, error) {
	queryParams := getPolkachuQueryParams(chainType)

	var response PolkachuChainAPIResponse
	httpClient := client.NewHTTPClient()
	_, err := httpClient.Get(fmt.Sprintf(PolkachuChainAPI, DefaultInitiaPolkachuName), "", queryParams, &response)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}

	if !response.PolkachuServices.StateSync.Active {
		return "", fmt.Errorf("state sync from polkachu is not active")
	}

	stateSyncURL := response.PolkachuServices.StateSync.Node
	if stateSyncURL == "" {
		return "", fmt.Errorf("no state sync URL found")
	}

	return stateSyncURL, nil
}

func FetchPolkachuStateSyncPeers(chainType registry.ChainType) (string, error) {
	queryParams := getPolkachuQueryParams(chainType)

	var response PolkachuPeersAPIResponse
	httpClient := client.NewHTTPClient()
	_, err := httpClient.Get(fmt.Sprintf(PolkachuPeersAPI, DefaultInitiaPolkachuName), "", queryParams, &response)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}

	stateSyncPeer := response.PolkachuPeer
	if stateSyncPeer == "" {
		return "", fmt.Errorf("no state-sync peer found")
	}

	return stateSyncPeer, nil
}

func FetchPolkachuPersistentPeers(chainType registry.ChainType) (string, error) {
	queryParams := getPolkachuQueryParams(chainType)

	var response PolkachuPeersAPIResponse
	httpClient := client.NewHTTPClient()
	_, err := httpClient.Get(fmt.Sprintf(PolkachuPeersAPI, DefaultInitiaPolkachuName), "", queryParams, &response)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}

	peers := response.LivePeers
	if len(peers) == 0 {
		return "", fmt.Errorf("no peers found")
	}

	return strings.Join(peers, ","), nil
}

func getPolkachuAddrBookEndpoint(chainType registry.ChainType) (string, error) {
	var networkPrefix string
	switch chainType {
	case registry.InitiaL1Testnet:
		networkPrefix = "testnet-"
	case registry.InitiaL1Mainnet:
		networkPrefix = ""
	default:
		return "", fmt.Errorf("chain type not supported: %v", chainType)
	}

	return fmt.Sprintf(PolkachuAddrBookAPI, networkPrefix, DefaultInitiaPolkachuName), nil
}

func DownloadPolkachuAddrBook(chainType registry.ChainType, dest string) error {
	addrBookEndpoint, err := getPolkachuAddrBookEndpoint(chainType)
	if err != nil {
		return err
	}

	httpClient := client.NewHTTPClient()
	err = httpClient.DownloadFile(addrBookEndpoint, dest, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to download addrbook: %w", err)
	}

	return nil
}
