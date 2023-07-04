package secure

// p2p packages implements the networking solution for the swell protocol.
// It connects every node of the network to each other, and establishes rules
// in order to

// The architecture consists of peer nodes, gateways and block listeners.
//
// Peer nodes comprises not only validators on the current checkpoint window
// but anyone who has established a connections and has minimum deposited stakes
// corresponding to their token. This includes nodes that are on synchronization
// process.
//
// Those in charge of proposing new blocks send them first to validating nodes.
// And then to all remaining nodes. TODO: create proxy facility
//
// Proposed blocks are send also to block listeners.
//
// Validators brodcast their veredict to all peer nodes and their block listeners
// Nodes receiving validatting veredicts forwards them to their block listeners.
//
// Non-proposing nodes inform block listeners about new blocks. They decide if
// they want to download blocks from you or not.
//
// Gateways send new events to the validating network. Every node should only
// subscribe to trusted gateways since there is no provision for DDoS attack
// protection. Gateways are supposed to behave honestly, respecting negotiated
// limmits.
