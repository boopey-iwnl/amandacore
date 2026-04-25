set(AMANDACORE_SHARED_ROOT ${CMAKE_CURRENT_SOURCE_DIR})

add_library(AmandaCoreShared STATIC
    ${AMANDACORE_SHARED_ROOT}/Source/GameplayRules.cpp
    ${AMANDACORE_SHARED_ROOT}/Source/ContentValidation.cpp
)

target_include_directories(AmandaCoreShared
    PUBLIC
        ${AMANDACORE_SHARED_ROOT}/Include
)

target_compile_features(AmandaCoreShared PUBLIC cxx_std_20)

if(MMO_REPLICA_ENABLE_TESTS)
    add_executable(AmandaCoreSharedTests
        ${AMANDACORE_SHARED_ROOT}/Tests/AmandaCoreSharedTests.cpp
    )

    target_link_libraries(AmandaCoreSharedTests PRIVATE AmandaCoreShared)
    add_test(NAME AmandaCoreSharedTests COMMAND AmandaCoreSharedTests)
endif()
