# Synthetic Track

CLI tool to track Synthetic.ai usage and leftover requests.

## API
### Quotas

``` bash
GET https://api.synthetic.new/v2/quotas
```



Retrieve information about your API usage quotas and limits.
Tip

/quotas requests do not count against your subscription limits!
The Synthetic API is still under development.

If there is more data you would like to see and access, please let us know!
Example Request

```bash
curl https://api.synthetic.new/v2/quotas \
  -H "Authorization: Bearer ${SYNTHETIC_API_KEY}"
```
Example Response

```json 

{
  "subscription": {
    "limit": 135,
    "requests": 0,
    "renewsAt": "2025-09-21T14:36:14.288Z"
  }
}
```

## Keep track 
Now that we know how to grab the usage and leftover requests, we can track them over time. 
I want to grab the usage every 30 minutes.

This way we can keep a history of the usage and respond to requests about historic usage and overall % usage over more time than the 4 hour window Synthetic keeps track of. 
