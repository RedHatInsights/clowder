import sys
from os.path import abspath
from os.path import dirname
from os.path import join

import sphinx_rtd_theme

# See: https://www.sphinx-doc.org/en/master/usage/configuration.html
#
#
# -- Path setup -----------------------------------------------------------------------------------
#
# If extensions (or modules to document with autodoc) are in another directory, add these
# directories to sys.path here. If the directory is relative to the documentation root, use
# os.path.abspath to make it absolute, like shown here.
#
# import os
# import sys
# sys.path.insert(0, os.path.abspath("."))
#
#
# -- Project information --------------------------------------------------------------------------

author = "Red Hat"
copyright = "2020, Red Hat"
project = "clowder"
release = "0.3.0"  # The full version, including alpha/beta/rc tags

# -- General configuration ------------------------------------------------------------------------

# List of patterns, relative to source directory, that match files and directories to ignore when
# looking for source files. This pattern also affects html_static_path and html_extra_path.
exclude_patterns = ["_build"]
nitpicky = True


# -- Options for HTML output ----------------------------------------------------------------------
html_logo = "images/clowder.svg"
html_theme = "sphinx_rtd_theme"
html_theme_path = [sphinx_rtd_theme.get_html_theme_path()]

master_doc = 'index'

#html_static_path = ["_static"]
#html_favicon = "images/iqefavicon.ico"
sys.path.append(abspath(join(dirname(__file__), "_ext")))
