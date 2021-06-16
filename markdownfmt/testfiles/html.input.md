Title
=

<img src="docs/img/some.png" alt="Logo">

<img src="img/arch.jpg" class="img-fluid" alt="architecture overview" />


<!--- comment  --->

<p align="center"><img src="docs/img/some.png" alt="Logo"></p>

<!--- multi line 
comment

--->

<table>
<tbody>
<tr><th>Avoid 🔥[Link](../docs/something.png)</th></tr>
<tr><td>

```go
resp, err := http.Get("http://example.com/")
if err != nil {
    // handle...
}
defer runutil.CloseWithLogOnErr(logger, resp.Body, "close response")

scanner := bufio.NewScanner(resp.Body)
// If any error happens and we return in the middle of scanning
// body, we can end up with unread buffer, which
// will use memory and hold TCP connection!
for scanner.Scan() {
```

</td></tr>
            <tr><th>Better 🤓</th></tr>
</tbody>
</table>

<dsada

<taasdav>
                  </taasdav>