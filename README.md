# Requirements and how to run
uuid package
	To install run "go get github.com/google/uuid" in a terminal
To run, run "go run ./ReceiptProcessor.go" in a terminal

# Sending API requests
* The server is listening on port 8080. 
	Issue the API requests like localhost:8080/receipts

## Receipts
### A receipt is invalid if:
* Any price is empty, null, or not a valid dollar amount.
	* If the price is missing completely, that is okay.

### A receipt will still be processed if:
* The date is not in the format "yyyy-MM-dd"
	* The date points will not be included if the format is incorrect
* The time is not in the format "hh:mm"
	* The time points will not be included if the format is incorrect
	* The time should be in 24 hour format to be processed correctly.
* e.g. 
	If the name of the retailer is not part of the receipt json it will still be processed.
	If the JSON is empty, the score is 0
