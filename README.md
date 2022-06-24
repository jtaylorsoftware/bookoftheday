# bookoftheday

Serverless NYT Best Sellers book of the day service. It allows users to subscribe with their email and receive a random book from their selected topics daily. It also provides
an endpoint to query the selected daily book for every topic available.

## Architecture

The service uses Lambda, Step Functions, SQS Queues, EventBridge scheduled rules, DynamoDB, and SES. It also exposes some endpoints (for subscribing and querying current books) via API Gateway. All Lambda function handlers are currently implemented using Go.

### Email Service

The email service operates by using two EventBridge scheduled rules - one weekly rule to refresh the NYT Best Seller List cache, and one daily rule to query the NYT Books API for a random book on a random date for each list.

#### Weekly Rule (Refresh lists)

The weekly rule just triggers the `RefreshLists` Lambda, which queries the `/lists/names.json` endpoint for the most up-to-date lists data and caches them using DynamoDB.

#### Daily Rule (Send emails)

The daily rule triggers the Step Functions state machine to run, which handles the process of obtaining random books. The state machine performs the following operations:

1. It invokes the `GetLists` Lambda and passes through the output as-is.
2. It uses a `Map` state to invoke the `GenerateRandomBooks` Lambda for each Best Seller list data in the input from (1). This function calculates a random publication date for its input list name and queries `/lists/{date}/{list}.json` to get a random book on that date's list. It then saves the book to a DynamoDB table before returning it as its output. The `Map` state uses a `MaxConcurrency` of `1` and a `Wait` step to avoid being rate limited on the calls to the NYT Books API.
3. The `ReadContacts` Lambda uses SES v2 to get a list of subscribed contacts, pairs each contact with a random book from its input, and sends each pairing to the email SQS Queue.

The `SendEmail` Lambda has an SQS trigger for the email Queue. It uses SES to send an email with the contact's book data.
