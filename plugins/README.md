# Plugins (experimental)

Plugins are a way to add more complex functionality to Linkquisition, beyond just simple rule-based matching.

## [Unwrap](./unwrap/unwrap.go) -plugin

This is the first plugin that I have written for Linkquisition. I use it to "unwrap" the internal links (in my company)
that are "wrapped" inside Microsoft Defender (Evergreen) URL when they are passed via Microsoft Teams or Outlook.

Now, the actual Defender URL is a good feature, but it is not very useful when the links refer to internal resources,
such as GitLab, Jira, Confluence, whatnot. To leverage the actually safeguarding part of the Defender URL, the plugin
can (and should) be configured to only unwrap the known sources, and leave the rest to be opened via the Defender URL.

To enable the plugin and configure it to unwrap the Defender links from Microsoft Teams you can use the following
configuration:

```json
{
  "browsers": [
    {
      "name": "Firefox",
      "command": "firefox %u",
      "hidden": false,
      "source": "auto",
      "matches": [
        {
          "type": "regex",
          "value": "^https?://github\\.com/Strobotti/"
        }
      ]
    }
  ],
  "plugins": [
    {
      "path": "unwrap.so",
      "settings": {
        "requireBrowserMatchToUnwrap": true,
        "rules": [
          {
            "match": "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
            "parameter": "url"
          }
        ]
      }
    }
  ]
}
```

In the above example, the `requireBrowserMatchToUnwrap` -setting is set to `true`, which means that the plugin will only
unwrap the links if there is a matching browser-rule for that URL and all the rest of the URLs are opened with full
Evergreen-protected URL.

## [Terminus](./terminus/terminus.go) -plugin

This plugin can be used to resolve redirects before processing actual rules or showing the browser picker dialog to the
user.

There are no configuration options available for this plugin yet, so simply enable it by adding it to the config.json
as follows:

```json
{
  "browsers": [
    ...
  ],
  "plugins": [
    {
      "path": "terminus.so"
    }
  ]
}

```

## Developing plugins

As stated before, the plugins-feature is experimental, the API is not stable and therefore subject to change. However,
the plugin-interface is quite simple and should be easy to implement.

The plugin is a shared object file (`.so`) that is loaded by the main application. The plugin must implement the
[linkquisition.Plugin](../plugin.go) -interface and the only currently supported feature is the `ModifyUrl` -function.
The function is called just before the URL is matched against the browser-rules and should return the modified URL, or 
the original if no modification is needed.
