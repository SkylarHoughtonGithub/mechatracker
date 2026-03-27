---
layout: default
title: Architecture Overview
---

# Architecture Documentation

Below is an automatically generated list of all architectural documents:

<ul>
  {% for file in site.static_files %}
    {% if file.path contains '/docs/architecture/' and file.extname == '.md' %}
      {% unless file.path contains 'overview.md' %}
        <li>
          <a href="{{ file.path | relative_url }}">
            {{ file.basename | capitalize | replace: "-", " " }}
          </a>
        </li>
      {% endunless %}
    {% endif %}
  {% endfor %}
</ul>