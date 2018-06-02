import { HexaUIPage } from './app.po';

describe('hexa-ui App', () => {
  let page: HexaUIPage;

  beforeEach(() => {
    page = new HexaUIPage();
  });

  it('should display message saying app works', () => {
    page.navigateTo();
    expect(page.getParagraphText()).toEqual('app works!');
  });
});
