export default (path) => {
  return `glob:${process.env.PHOENIX_BASEURL || 'http://localhost'}${path}*`
}
