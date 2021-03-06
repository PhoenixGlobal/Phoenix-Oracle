const extractBuildInfo = () => {
  const matches =
    /(\d+\.\d+\.\d+)@(.+)$/g.exec(process.env.PHOENIX_VERSION) || []
  return {
    version: matches[1] || 'unknown',
    sha: matches[2] || 'unknown',
  }
}

export default extractBuildInfo
