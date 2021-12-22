import { getAuthentication, setAuthentication } from '../../src/utils/storage'

describe('utils/storage', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('getAuthentication', () => {
    it('returns a JS object for JSON stored as "phoenix.authentication" in localStorage', () => {
      localStorage.setItem('phoenix.authentication', '{"allowed":true}')
      expect(getAuthentication()).toEqual({ allowed: true })
    })
  })

  describe('setAuthentication', () => {
    it('saves the JS object as JSON under the key "phoenix.authentication" in localStorage', () => {
      setAuthentication({ allowed: true })
      expect(localStorage.getItem('phoenix.authentication')).toEqual(
        '{"allowed":true}',
      )
    })
  })
})
