## API

### POST /create
Creates a new account

##### Response
```
  {
    "login": "string",
    "password": "string",
  }
```

### POST /auth
Get a new access token for the user

##### Request
```
  {
    "login": "string",
    "password": "string",
    "refresh_token": "string" // optional
  }
```

##### Response
```
  {
    "access_token": "string",
    "refresh_token": "string",
  }
```

### POST /addinvoice
Create a new invoice

##### Request
```
  {
    "amt": "string",
    "memo": "string", // optional
    "description_hash": "string" // optional
  }
```

##### Response
```
  {
    "r_hash": "string",
    "payment_request": "string",
    "pay_req": "string", // same as payment_request for compatibility
  }
```

### GET /balance
Get the user's balance

##### Response
```
  {
    "BTC": {
      "available_balance": number 
    }
  }
```
