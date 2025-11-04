# GraphQL Token Transfer API

Project providing a GraphQL API for transferring tokens between wallets, implemented using Go and PostgreSQL. \
The main focus was on eliminating database race conditions during concurrent token transfers.

## Technologies
- **Programming Language**: Go
- **GraphQL Framework**: gqlgen
- **Database**: PostgreSQL (Managed with GORM)
- **Containerization**: Docker, Docker Compose


## Data model
The project uses a single model, `Wallet`:

| Field | Type (Go) | Storage (PostgreSQL) | Constraint |
| :--- | :--- | :--- | :--- |
| `Address` | `string` | `VARCHAR(42)` | Unique (Format: `0x...`) |
| `Balance` | `int64` | `BIGINT` | Cannot go negative (enforced by logic) |

## Installation

### Prerequisites
- **Go**
- **Docker and Docker Compose**

### Clone the repository

```bash
git clone https://github.com/SkMarcin/Token-Transfer-API.git
cd Token-Transfer-API
```

### Configuration
Configuration variables should be stored in .env file, an example is provided in the repository.


```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=token_transfer_db
PORT=8080
```


## Testing
### Running Tests
Tests require a running database instance.
```bash
docker-compose up -d
go test ./...
```

### Concurrency testing
In the first implementation of Transfer, there was a possibility of race conditions occuring when two transfers from the same wallet were smaller than the balance separately but larger as a sum. This behavior can be observed in the branch **exp/concurrency-failed-test**, where balance can go into the negatives during a test utilizing sync.WaitGroup to cause a race condition.

Race conditions were eliminated by utilizing Pessimistic Locking during a transfer.


## GraphQL API Usage

The API exposes GraphQL endpoints for requesting transfers between wallets.

### Running the API

```bash
docker-compose up -d
go run ./cmd/api/main.go
```

The API can be accessed at http://localhost:8080.

### Example Mutations
```graphql
mutation SuccessfulTransfer {
  transfer(
    fromAddress: "0x0000000000000000000000000000000000000000",
    toAddress: "0x1111111111111111111111111111111111111111",
    amount: 100
  )
}
```
Expected Result (on first run): {"data": {"transfer": 999900}}


```graphql
mutation InsufficientBalanceFailure {
  transfer(
    fromAddress: "0x0000000000000000000000000000000000000000",
    toAddress: "0x2222222222222222222222222222222222222222",
    amount: 5000000
  )
}
```
Expected Result (Error): The response will contain a message: "Insufficient balance".
