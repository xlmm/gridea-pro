const generateOverride = (params = {}) => {
    let result = ''

    // Primary color
    if (params.primaryColor && params.primaryColor !== '#e8590c') {
        result += `
      :root {
        --primary-color: ${params.primaryColor};
      }
    `
    }

    // Accent color
    if (params.accentColor && params.accentColor !== '#D97706') {
        result += `
      :root {
        --color-accent: ${params.accentColor};
      }
    `
    }

    // Max content width
    if (params.maxWidth && params.maxWidth !== '720px') {
        result += `
      :root {
        --content-max-width: ${params.maxWidth};
      }
    `
    }

    // Font family
    if (params.fontFamily) {
        let fontStack = ''
        switch (params.fontFamily) {
            case 'serif':
                fontStack = "'Noto Serif SC', Georgia, serif"
                break
            case 'Georgia':
                fontStack = "Georgia, 'Noto Serif SC', serif"
                break
            case 'Noto Serif SC':
                fontStack = "'Noto Serif SC', serif"
                break
            case 'sans-serif':
                fontStack = "Inter, -apple-system, sans-serif"
                break
            default:
                fontStack = "-apple-system, BlinkMacSystemFont, 'Segoe UI', Inter, sans-serif"
        }
        result += `
      :root {
        --font-body: ${fontStack};
      }
      body {
        font-family: ${fontStack};
      }
    `
    }

    // Custom CSS
    if (params.customCss) {
        result += `
      ${params.customCss}
    `
    }

    return result
}

module.exports = generateOverride
