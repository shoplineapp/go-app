# Redactor
The redactor allows you to redact certain sensitive information by two modes: full/redact.
The normal redactor will redact following fields:
- password
- cipher
- email
- address
## How to use
### Default Redactor
```
common.DefaultRedactor.Redact(request)
```
### Create your own
```
r := &common.Redactor{}
r.Redact(request)
```
### Add filters
*By general field*
This will redact all field with `fieldName` in a map or any struct
```
r.AddFilter(NewFilter("nil", "fieldName", common.FullRedact))
```
*By Struct*
This will redact the only `ABC` struct
```
type ABC struct {}
r.AddFilter(NewFilter(ABC{}, "", common.FullRedact))
```

*By struct's field*
This will redact only the `Name` field inside `ABC` struct
```
type ABC struct {
  Name string
}

r.AddFilter(NewFilter(ABC{}, "Name", common.FullRedact))
```
### Specificity
Struct filter > struct's field > general field