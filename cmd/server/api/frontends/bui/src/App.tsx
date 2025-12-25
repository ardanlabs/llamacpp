import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import ModelList from './components/ModelList';
import ModelPs from './components/ModelPs';
import ModelPull from './components/ModelPull';
import CatalogList from './components/CatalogList';
import LibsPull from './components/LibsPull';
import SecurityKeyList from './components/SecurityKeyList';
import SecurityKeyCreate from './components/SecurityKeyCreate';
import SecurityKeyDelete from './components/SecurityKeyDelete';
import SecurityTokenCreate from './components/SecurityTokenCreate';
import DocsSDK from './components/DocsSDK';
import DocsSDKKronk from './components/DocsSDKKronk';
import DocsSDKModel from './components/DocsSDKModel';
import DocsSDKExamples from './components/DocsSDKExamples';
import DocsCLI from './components/DocsCLI';
import DocsWebAPI from './components/DocsWebAPI';
import { ModelListProvider } from './contexts/ModelListContext';

export type Page =
  | 'home'
  | 'model-list'
  | 'model-ps'
  | 'model-pull'
  | 'catalog-list'
  | 'libs-pull'
  | 'security-key-list'
  | 'security-key-create'
  | 'security-key-delete'
  | 'security-token-create'
  | 'docs-sdk'
  | 'docs-sdk-kronk'
  | 'docs-sdk-model'
  | 'docs-sdk-examples'
  | 'docs-cli'
  | 'docs-webapi';

export const routeMap: Record<Page, string> = {
  'home': '/',
  'model-list': '/models',
  'model-ps': '/models/running',
  'model-pull': '/models/pull',
  'catalog-list': '/catalog',
  'libs-pull': '/libs/pull',
  'security-key-list': '/security/keys',
  'security-key-create': '/security/keys/create',
  'security-key-delete': '/security/keys/delete',
  'security-token-create': '/security/tokens/create',
  'docs-sdk': '/docs/sdk',
  'docs-sdk-kronk': '/docs/sdk/kronk',
  'docs-sdk-model': '/docs/sdk/model',
  'docs-sdk-examples': '/docs/sdk/examples',
  'docs-cli': '/docs/cli',
  'docs-webapi': '/docs/webapi',
};

export const pathToPage: Record<string, Page> = Object.fromEntries(
  Object.entries(routeMap).map(([page, path]) => [path, page as Page])
);

function HomePage() {
  return (
    <div className="welcome">
      <img
        src="https://raw.githubusercontent.com/ardanlabs/kronk/refs/heads/main/images/project/kronk_banner.jpg"
        alt="Kronk Banner"
        className="welcome-banner"
      />
      <h2>Welcome to Kronk</h2>
      <p>Select an option from the sidebar to manage your Kronk environment.</p>
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <ModelListProvider>
        <Layout>
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/models" element={<ModelList />} />
            <Route path="/models/running" element={<ModelPs />} />
            <Route path="/models/pull" element={<ModelPull />} />
            <Route path="/catalog" element={<CatalogList />} />
            <Route path="/libs/pull" element={<LibsPull />} />
            <Route path="/security/keys" element={<SecurityKeyList />} />
            <Route path="/security/keys/create" element={<SecurityKeyCreate />} />
            <Route path="/security/keys/delete" element={<SecurityKeyDelete />} />
            <Route path="/security/tokens/create" element={<SecurityTokenCreate />} />
            <Route path="/docs/sdk" element={<DocsSDK />} />
            <Route path="/docs/sdk/kronk" element={<DocsSDKKronk />} />
            <Route path="/docs/sdk/model" element={<DocsSDKModel />} />
            <Route path="/docs/sdk/examples" element={<DocsSDKExamples />} />
            <Route path="/docs/cli" element={<DocsCLI />} />
            <Route path="/docs/webapi" element={<DocsWebAPI />} />
          </Routes>
        </Layout>
      </ModelListProvider>
    </BrowserRouter>
  );
}

export default App;
