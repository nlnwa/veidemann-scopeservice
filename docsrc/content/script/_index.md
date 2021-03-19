---
title: "Script"
date: 2021-03-17T14:11:32+01:00
---

Scope scripts are written in [Starlark](https://github.com/bazelbuild/starlark/blob/master/spec.md), a dialect of Python.
In most cases it is not necessary to know Python or Starlark to write Scope scripts because we hav built in a number of 
functions which do most of the heavy lifting.

Example:
```
isScheme(param('scope_allowedSchemes')).otherwise(Blocked)
isSameHost(param('scope_includeSubdomains'), altSeeds=param('scope_altSeeds')).then(Include, continueEvaluation=True).otherwise(Blocked, continueEvaluation=False)
maxHopsFromSeed(param('scope_maxHopsFromSeed'), param('scope_hopsIncludeRedirects')).then(TooManyHops)
isUrl(param('scope_excludedUris')).then(Blocked)
```

{{< highlight py3 "linenos=inline" >}}
isScheme(param('scope_allowedSchemes')).otherwise(Blocked)
isSameHost(param('scope_includeSubdomains'), altSeeds=param('scope_altSeeds')).then(Include, continueEvaluation=True).otherwise(Blocked, continueEvaluation=False)
maxHopsFromSeed(param('scope_maxHopsFromSeed'), param('scope_hopsIncludeRedirects')).then(TooManyHops)
isUrl(param('scope_excludedUris')).then(Blocked)
{{< / highlight >}}
