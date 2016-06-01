# `combos` - Parser Combinators

by Tim Henderson (tadh@case.edu)

Copyright 2016, Licensed under the GPL version 3. Please reach out to me
directly if you require another licensing option. I am willing to work with you.

[![GoDoc](https://godoc.org/github.com/timtadh/combos?status.svg)](https://godoc.org/github.com/timtadh/combos)

## Why?

Because I can. I have written a lot of recursive descent parsers over the years.
Normally I write them when I am in an environment where I can't pull in an
external library or a code generation system like `yacc`. Usually I also hand
write a lexer as well by manually drawing out the NFA converting it to a DFA and
"compiling" it by hand.  All of this is very tedious, but it does make one good
at top down parsing.

These days when I implement a recursive descent parser the first thing I do is
implement a few basic *combinator* functions. I actually discovered how to do
this on my own before I had read the
[literature](https://en.wikipedia.org/wiki/Parser_combinator) on the subject so
my combinators may be a little different than the normal ones. Here they are:

1. `epsilon` - consume zero tokens. Always matches.
2. `consume` - consume one token of the given name. Error if the token was not
   of the expected name or we are at EOS (end of string).
3. `concat` - concatentate 2 or more combinators together. I usually call my
   combinators "consumers" in the code as that is what they do, consume
   text/tokens.
4. `alt` - match of the given alternative combinators. I usually implement this
   as an ordered alternative where the first matching combinator is used. In the
   past I have also implemented longest match alternatives but the performance
   of such an operator is usually really bad.

That's it! All you need to write a recursive descent parser. Any other function
can constructed from these functions. For instance, the `maybe` function (as in
maybe consume the tokens matched by the combinator).

```
maybe(c : consumer) = alt(c, epsilon)
```

I decided to make the parser combinators I usually use in Go and make them
available as a library. There are other parser combinator libraries for Go but
this one is mine.

## Dependencies

1. [`lexmachine`](https://github.com/timtadh/lexmachine). A lexical analysis
   framework I also wrote, use it to write the lexer.

