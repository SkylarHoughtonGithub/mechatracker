---
layout: default
title: Architecture Overview
---

# Architecture Documentation

Below is an automatically generated list of all architectural documents:

<ul>
  {% for p in site.pages %}
    {% if p.path contains 'docs/architecture/' %}
      {% unless p.url == page.url %}
        <li>
          <a href="{{ p.url | relative_url }}">
            {{ p.title | default: p.basename }}
          </a>
        </li>
      {% endunless %}
    {% endif %}
  {% endfor %}
</ul>