package script

import (
	"fmt"
	"go.starlark.net/starlark"
	"strings"
)

func init() {
	starlark.Universe["test"] = starlark.NewBuiltin("test", test)
	starlark.Universe["isScheme"] = starlark.NewBuiltin("isScheme", isScheme)
	starlark.Universe["isSameHost"] = starlark.NewBuiltin("isSameHost", isSameHost)
	starlark.Universe["maxHopsFromSeed"] = starlark.NewBuiltin("maxHopsFromSeed", maxHopsFromSeed)
	starlark.Universe["isUrl"] = starlark.NewBuiltin("isUrl", isUrl)
}

func test(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var m starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "match", &m); err != nil {
		return nil, err
	}
	match := Match(parameterAsBool(m))
	printDebug(thread, b, args, kwargs, fmt.Sprintf("match=%v", match))
	return match, nil
}

func isScheme(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var scheme string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &scheme); err != nil {
		return nil, err
	}
	qUrl := thread.Local(urlKey).(*UrlValue)
	s := strings.TrimRight(qUrl.parsedUri.Protocol(), ":")
	scheme = strings.ToLower(scheme)
	match := False
	for _, t := range strings.Fields(scheme) {
		if t == s {
			match = True
			break
		}
	}

	printDebugf(thread, b, args, kwargs, "scheme=%v, wantScheme=%v, match=%v", s, scheme, match)

	return match, nil
}

func isSameHost(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var includeSubdomains starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "includeSubdomains?", &includeSubdomains); err != nil {
		return nil, err
	}

	match := false
	qUrl := thread.Local(urlKey).(*UrlValue)
	host := qUrl.parsedUri.Hostname()
	if seed, err := ScopeCanonicalizationProfile.Parse(qUrl.qUri.SeedUri); err == nil {
		match = host == seed.Hostname()
		if !match && parameterAsBool(includeSubdomains) {
			match = strings.HasSuffix(host, "."+seed.Hostname())
		}
		printDebugf(thread, b, args, kwargs, "host=%v, seedHost=%v, match=%v", host, seed.Hostname(), match)
	} else {
		printDebugf(thread, b, args, kwargs, "Could not parse seed '%v'", qUrl.qUri.SeedUri)
		return nil, IllegalUri.asError(fmt.Sprintf("Could not parse seed '%v'", qUrl.qUri.SeedUri))
	}

	return Match(match), nil
}

func maxHopsFromSeed(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var maxHops starlark.Value
	var includeRedirects starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "hops", &maxHops, "includeRedirects?", &includeRedirects); err != nil {
		return nil, err
	}
	qUrl := thread.Local(urlKey).(*UrlValue)
	discoveryPath := qUrl.qUri.GetDiscoveryPath()
	if !parameterAsBool(includeRedirects) {
		discoveryPath = strings.ReplaceAll(discoveryPath, "R", "")
	}

	var match bool

	if h, err := parameterAsInt64(maxHops); err == nil {
		match = len(discoveryPath) >= int(h)
	} else {
		if err != None {
			return nil, err
		}
	}
	printDebugf(thread, b, args, kwargs, "discoveryPath=%v, hops=%v, match=%v", discoveryPath, len(discoveryPath), match)
	return Match(match), nil
}

func isUrl(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var u string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "url", &u); err != nil {
		return nil, err
	}
	qUrl := thread.Local(urlKey).(*UrlValue)

	match := False
	for _, ux := range strings.Fields(u) {
		canon, err := ScopeCanonicalizationProfile.Parse(ux)
		if err != nil {
			return nil, err
		}
		ux = canon.String()
		if qUrl.String() == ux {
			match = True
			break
		}
	}

	printDebugf(thread, b, args, kwargs, "test='%v', url=%v, match=%v", u, qUrl.String(), match)

	return match, nil
}