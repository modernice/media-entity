import { defineBuildConfig } from 'unbuild'
import pkg from './package.json'

export default defineBuildConfig({
  declaration: true,
  entries: [
    'src/index',
    {
      input: 'src/entities/gallery',
      name: 'entities/gallery',
    },
  ],

  rollup: {
    emitCJS: true,
  },

  externals: Object.keys(pkg.dependencies),
})
