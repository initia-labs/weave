package types

type Channel struct {
	PortID    string `json:"port_id"`
	ChannelID string `json:"channel_id"`
}

type IBCChannelPair struct {
	L1ConnectionID string
	L1             Channel
	L2ConnectionID string
	L2             Channel
}

// ChannelResponse define a minimal struct to parse just the counterparty field
type ChannelResponse struct {
	Channel struct {
		ConnectionHops []string `json:"connection_hops"`
		Counterparty   Channel  `json:"counterparty"`
	} `json:"channel"`
}

type ChannelsResponse struct {
	Channels []struct {
		PortID         string   `json:"port_id"`
		ChannelID      string   `json:"channel_id"`
		ConnectionHops []string `json:"connection_hops"`
		Counterparty   Channel  `json:"counterparty"`
	} `json:"channels"`
}
