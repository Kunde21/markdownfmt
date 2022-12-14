# An h1 header

Paragraphs are separated by a blank line.

2nd paragraph. *Italic*, **bold**, `monospace`.

Trying items.

1. Item 1
2. Item 2
   1. Item 2a
      - Item 2aa
   2. Item 2b
3. Item 3

> Block quotes are written like so.
>
> There is a bug in blackfriday preventing code inside blockquotes.
>
> > They can be nested.
>
> They can span multiple paragraphs, if you like.

Last paragraph here.

## An h2 header

- Paragraph right away.
- **Big item**: Right away after header.

[Visit GitHub!](www.github.com)

~~Mistaken text.~~

This (**should** be *fine*).

A \> B.

It's possible to backslash escape \<html\> tags and \`backticks\`. They are treated as text.

1986\. What a great season.

The year was 1986. What a great season.

\*literal asterisks\*.

---

[http://example.com](http://example.com)

Now a [link](www.github.com) in a paragraph. End with [link_underscore.go](www.github.com).

- [Link](www.example.com)

### An h3 header

Here's a numbered list:

1. first item
2. second item
3. third item

Code block

```
define foobar() {
    print "Welcome to flavor country!";
}
```

With language

```Go
func main() {
	println("Hi.")
}
```

With language and some tags.

```Go some tags = whatever, but should be preserved.
func main() {
	println("Hi.")
}
```

Here's a table.

| Name  | Age |
|-------|-----|
| Bob   | 27  |
| Alice | 23  |

Colons can be used to align columns.

| Tables        | Are           | Cool      |
|---------------|:-------------:|----------:|
| col 3 is      | right-aligned |     $1600 |
| col 2 is      |   centered!   |       $12 |
| zebra stripes |   are neat    |        $1 |
| support for   | サブタイトル  | priceless |

The outer pipes (|) are optional, and you don't need to make the raw Markdown line up prettily. You can also use inline Markdown.

| Markdown | More      | Pretty     |
|----------|-----------|------------|
| *Still*  | `renders` | **nicely** |
| 1        | 2         | 3          |

# Nested Lists

### Codeblock within list

- Code block in list does not work reliably

Para

### Blockquote within list

- list1

  > This a quote within a list.
  >
  > Still going  
  > with broken line

### Table within list

- list1

  | Header One | Header Two |
  |------------|------------|
  | Item One   | Item Two   |

### Multi-level nested

- Item 1

  Another paragraph inside this list item is indented just like the previous paragraph.

- Item 2

  - Item 2a

    Things go here.

    > This a quote within a list.

    And they stay here.

  - Item 2b

- Item 3

# Line Breaks

Some text with two trailing spaces for linebreak.  
More text immediately after.  
Useful for writing poems.

[Link](path\\to\\page)

![Markdown Format Demo](https://github.com/shurcooL/atom-markdown-format/blob/master/Demo.gif?raw=true)

[https://path\to\page](https://path\\to\\page)

Links in Markdown are not changed, like http://google.com

Done.
