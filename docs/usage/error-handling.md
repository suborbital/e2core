# Error handling

When building your Atmo app, handling errors returned from Runnables is pretty essential. When a Runnable returns an error, it contains a `code` and a `message`. The `code` must be a valid [HTTP response status code](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status). Using the Directive, you can manage how your application behaves when an error is returned:

{% hint style="info" %}
The default behaviour for any error is for the handler to return.
{% endhint %}

Any time a Runnable returns an error, you can decide what to do with it using the `onErr` clause:

```yaml
- type: request
    resource: /repo/report/*repo
    method: GET
    steps:
      - fn: check-cache
        as: report
        onErr:
          any: continue

      - fn: send-report
```

In its basic form, onErr allows you to tell Atmo to ignore any error from a Runnable. When using `continue`, the JSON of the error will be placed into state, such as `{"code":404,"message":"not found"}`

To gain more control, you can choose what to do based on error codes:

```yaml
- type: request
    resource: /repo/report/*repo
    method: GET
    steps:
      - fn: check-cache
        as: report
        onErr:
          code:
            404: continue
          other: return

      - fn: send-report
```

Technically, any `return` \(such as an 'any' or 'other'\) can be omitted since it is the default behaviour, but it can improve readability of your Directive when included.

When defining specific error codes, you cannot use 'any', use 'other' instead. If no specific codes are specified, use 'any'.

Whenever Atmo decides to return based on your Directive's instructions, the error's JSON is returned to the caller with the status code set to the error code.

