import React, { FC, useState } from 'react'
import Box from '@mui/material/Box'

import Page from '../components/system/Page'
import Toolbar from '../components/create/Toolbar'
import ConfigWindow from '../components/create/ConfigWindow'
import SessionTypeTabs from '../components/create/SessionTypeTabs'
import SessionTypeSwitch from '../components/create/SessionTypeSwitch'
import CenterMessage from '../components/create/CenterMessage'
import ExamplePrompts from '../components/create/ExamplePrompts'
import InferenceTextField from '../components/create/InferenceTextField'
import Disclaimer from '../components/widgets/Disclaimer'

import useRouter from '../hooks/useRouter'
import useLightTheme from '../hooks/useLightTheme'
import useCreateInputs from '../hooks/useCreateInputs'

import {
  ISessionMode,
  ISessionType,
  ICreateSessionConfig,
  SESSION_MODE_INFERENCE,
  SESSION_MODE_FINETUNE,
  SESSION_TYPE_TEXT,
} from '../types'

import {
  DEFAULT_SESSION_CONFIG,
  HELIX_DEFAULT_TEXT_MODEL,
} from '../config'

const PADDING_X = 3

const Create: FC = () => {
  const router = useRouter()
  const lightTheme = useLightTheme()
  const inputs = useCreateInputs()

  const [ sessionConfig, setSessionConfig ] = useState<ICreateSessionConfig>(DEFAULT_SESSION_CONFIG)
  const [ showConfigWindow, setShowConfigWindow ] = useState(false)

  const mode = (router.params.mode as ISessionMode) || SESSION_MODE_INFERENCE
  const type = (router.params.type as ISessionType) || SESSION_TYPE_TEXT
  const model = router.params.model || HELIX_DEFAULT_TEXT_MODEL

  const topbar = (
    <Toolbar
      mode={ mode }
      type={ type }
      model={ model }
      onOpenConfig={ () => setShowConfigWindow(true) }
      onSetMode={ mode => router.setParams({mode}) }
      onSetModel={ model => router.setParams({model}) }
    />
  )

  const headerContent = mode == SESSION_MODE_FINETUNE && (
    <SessionTypeTabs
      type={ type }
      onSetType={ type => router.setParams({type}) }
    />
  )

  const footerContent = mode == SESSION_MODE_INFERENCE ? (
    <Box sx={{ px: PADDING_X }}>
      <Box sx={{ mb: 3 }}>
        <ExamplePrompts
          type={ type }
          onChange={ (prompt) => {
            inputs.setInputValue(prompt)
          }}
        />
      </Box>
      <Box sx={{ mb: 1 }}>
        <InferenceTextField
          type={ type }
          value={ inputs.inputValue }
          disabled={ mode == SESSION_MODE_FINETUNE }
          startAdornment={(
            <SessionTypeSwitch
              type={ type }
              onSetType={ type => router.setParams({type}) }
            />
          )}
          onUpdate={ inputs.setInputValue }
          onInference={ () => {
            console.log('--------------------------------------------')
            console.log(inputs.inputValue)
          }}
        />
      </Box>
      <Box sx={{ mb: 1 }}>
        <Disclaimer />
      </Box>
    </Box>
  ) : (
    <Box
      sx={{
        p: 0,
        borderTop: lightTheme.border,
        height: '100px',
        backgroundColor: 'rgba(0,0,0,0.5)',
      }}
    >

    </Box>
  )

  return (
    <Page
      topbarTitle={ mode == SESSION_MODE_FINETUNE ? 'The start of something beautiful' : '' }
      topbarContent={ topbar }
      headerContent={ headerContent }
      footerContent={ footerContent }
      px={ PADDING_X }
      sx={{
        backgroundImage: lightTheme.isLight ? 'url(/img/nebula-light.png)' : 'url(/img/nebula-dark.png)',
        backgroundSize: '80%',
        backgroundPosition: mode == SESSION_MODE_INFERENCE ? 'center center' : `center ${window.innerHeight - 280}px`,
        backgroundRepeat: 'no-repeat',
      }}
    >
      
      {
        mode == SESSION_MODE_INFERENCE && (
          <>
            <Box
              sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                pt: 10,
              }}
            >
              <CenterMessage
                type={ type }
                onSetType={ type => router.setParams({type}) }
              />
            </Box>
          </>
        )
      }
      {
        showConfigWindow && (
          <ConfigWindow
            mode={ mode }
            type={ type }
            sessionConfig={ sessionConfig }
            onSetSessionConfig={ setSessionConfig }
            onClose={ () => setShowConfigWindow(false) }
          />
        )
      }
    </Page>
  )
}

export default Create
