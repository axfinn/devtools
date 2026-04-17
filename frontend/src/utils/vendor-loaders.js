let mermaidPromise
let lastMermaidConfigKey = ''

export async function getMermaid(config = {}) {
  const mod = await (mermaidPromise ||= import('mermaid'))
  const mermaid = mod.default ?? mod
  const mergedConfig = {
    startOnLoad: false,
    securityLevel: 'loose',
    ...config
  }
  const configKey = JSON.stringify(mergedConfig)

  if (configKey !== lastMermaidConfigKey) {
    mermaid.initialize(mergedConfig)
    lastMermaidConfigKey = configKey
  }

  return mermaid
}

let echartsPromise

export async function getECharts() {
  return echartsPromise ||= Promise.all([
    import('echarts/core'),
    import('echarts/charts'),
    import('echarts/components'),
    import('echarts/renderers')
  ]).then(([core, charts, components, renderers]) => {
    core.use([
      charts.LineChart,
      charts.BarChart,
      charts.PieChart,
      components.GridComponent,
      components.LegendComponent,
      components.TooltipComponent,
      components.VisualMapComponent,
      renderers.CanvasRenderer
    ])
    return core
  })
}
