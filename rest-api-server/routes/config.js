const NETWORK_DIR = '/home/admin/st-poc-hyperledger/hyperledger/sto-test-network';
const ORG_CONFIG = {
    "orgConfig1": {
        "CryptoPath": `${NETWORK_DIR}/organizations/peerOrganizations/org1.example.com`,
        "PeerEndpoint": "localhost:7051",
        "TLSCertPath": "/peers/peer0.org1.example.com/tls/ca.crt"
    },
    "orgConfig2": {
        "CryptoPath": `${NETWORK_DIR}/organizations/peerOrganizations/org2.example.com`,
        "PeerEndpoint": "localhost:9051",
        "TLSCertPath": "/peers/peer0.org2.example.com/tls/ca.crt"
    },
    "ordererConfig": {
        "CryptoPath": `${NETWORK_DIR}/organizations/ordererOrganizations/example.com`,
        "PeerEndpoint": "localhost:7050",
        "TLSCertPath": "/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com"
    }
}

module.exports = {
    NETWORK_DIR, 
    ORG_CONFIG
};